package v1

import (
	"errors"
	"strconv"

	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/user"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserRouter struct {
	userService user.Service
}

func NewUserRouter(userService user.Service) *UserRouter {
	return &UserRouter{
		userService: userService,
	}
}

func InitUserRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	auditDao := audit.NewDao(db)
	auditService := audit.NewService(auditDao)

	userDao := user.NewDao(db)
	userService := user.NewService(userDao, jwtSecret, auditService)
	userRouter := NewUserRouter(userService)

	userGroup := r.Group("")
	userGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		userGroup.GET("/me", userRouter.GetCurrentUser)
		userGroup.GET("/users", userRouter.GetUserByID)
		userGroup.GET("/users/count", userRouter.GetUserCount)
		userGroup.GET("/users/list", userRouter.GetAllUsers)
		userGroup.POST("/approve_user", userRouter.ApproveUser)
		userGroup.POST("/reject_user", userRouter.RejectUser)
		userGroup.POST("/set_user_role", userRouter.SetUserRole)
		userGroup.POST("/avatar", userRouter.UpdateAvatar)
	}
}

func (r *UserRouter) GetCurrentUser(c *gin.Context) {
	// 1. 从中间件存入的 context 中获取当前用户 ID
	id, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized", "code": constant.ErrorCode})
		return
	}

	// 2. 根据 ID 查询用户信息
	user, err := r.userService.GetUserByID(c.Request.Context(), id.(string))
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	// 3. 返回用户信息数据（注意这里要包在 data 字段里，保持前端统一性）
	c.JSON(200, gin.H{
		"message": "获取成功",
		"data":    user,
		"code":    constant.SuccessCode,
	})
}

func (r *UserRouter) GetUserByID(c *gin.Context) {
	id := c.Query("id")
	user, err := r.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"user": user, "code": constant.SuccessCode})
}

// GetUserCount 获取用户统计信息
func (r *UserRouter) GetUserCount(c *gin.Context) {
	// 获取总用户数
	totalCount, err := r.userService.GetTotalUserCount(c.Request.Context())
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	// 获取各状态的用户数
	activeCount, _ := r.userService.GetUserCountByStatus(c.Request.Context(), constant.UserStatusActive)
	inactiveCount, _ := r.userService.GetUserCountByStatus(c.Request.Context(), constant.UserStatusInactive)
	lockedCount, _ := r.userService.GetUserCountByStatus(c.Request.Context(), constant.UserStatusLocked)

	c.JSON(200, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"total":    totalCount,
			"active":   activeCount,
			"inactive": inactiveCount,
			"locked":   lockedCount,
		},
		"code": constant.SuccessCode,
	})
}

// GetAllUsers 获取用户列表（分页）
func (r *UserRouter) GetAllUsers(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取用户列表
	users, total, err := r.userService.GetAllUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	c.JSON(200, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"users":       users,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
		"code": constant.SuccessCode,
	})
}

type UserActionRequest struct {
	ID string `json:"id" binding:"required"`
}

func (r *UserRouter) ApproveUser(c *gin.Context) {
	var req UserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	if err := r.userService.ApproveUser(c.Request.Context(), req.ID); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "User approved successfully", "code": constant.SuccessCode})
}

func (r *UserRouter) RejectUser(c *gin.Context) {
	var req UserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	if err := r.userService.RejectUser(c.Request.Context(), req.ID); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "User rejected successfully", "code": constant.SuccessCode})
}

type SetUserRoleRequest struct {
	TargetUserID string            `json:"target_user_id" binding:"required"`
	NewRole      constant.UserRole `json:"new_role" binding:"required"`
}

func (r *UserRouter) SetUserRole(c *gin.Context) {
	var req SetUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}

	// 从 Context 获取当前操作者 ID
	operatorIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized", "code": constant.ErrorCode})
		return
	}
	operatorID := operatorIDStr.(string)

	if err := r.userService.SetUserRole(c.Request.Context(), operatorID, req.TargetUserID, req.NewRole); err != nil {
		if errors.Is(err, constant.ErrUnauthorized) {
			c.JSON(403, gin.H{"error": "权限不足", "code": constant.ErrorCode})
			return
		}
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	c.JSON(200, gin.H{"message": "User role updated successfully", "code": constant.SuccessCode})
}

type UpdateAvatarRequest struct {
	Avatar string `json:"avatar" binding:"required"`
}

func (r *UserRouter) UpdateAvatar(c *gin.Context) {
	var req UpdateAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	// 1. 获取当前登录用户 ID
	id, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"message": "Unauthorized", "code": constant.ErrorCode})
		return
	}

	// 2. 更新头像
	if err := r.userService.UpdateAvatar(c.Request.Context(), id.(string), req.Avatar); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	c.JSON(200, gin.H{
		"message": "头像更新成功",
		"code":    constant.SuccessCode,
		"data": gin.H{
			"avatar": req.Avatar,
		},
	})
}
