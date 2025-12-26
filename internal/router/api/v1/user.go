package v1

import (
	"strconv"

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
	userDao := user.NewDao(db)
	userService := user.NewService(userDao, jwtSecret)
	userRouter := NewUserRouter(userService)

	userGroup := r.Group("")
	userGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		userGroup.GET("/users", userRouter.GetUserByID)
		userGroup.GET("/users/count", userRouter.GetUserCount)
		userGroup.GET("/users/list", userRouter.GetAllUsers)
	}
}

func (r *UserRouter) GetUserByID(c *gin.Context) {
	id := c.Query("id")
	user, err := r.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"user": user, "code": constant.SuccessCode})
}

// GetUserCount 获取用户统计信息
func (r *UserRouter) GetUserCount(c *gin.Context) {
	// 获取总用户数
	totalCount, err := r.userService.GetTotalUserCount(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
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
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
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
