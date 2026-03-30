package api

import (
	"fmt"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/auth"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/project"
	"helios-auth-service/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthRouter struct {
	authService    auth.Service
	projectService project.Service
	auditService   audit.Service
}

// NewAuthRouter 创建 AuthRouter 实例
func NewAuthRouter(authService auth.Service, projectService project.Service, auditService audit.Service) *AuthRouter {
	return &AuthRouter{
		authService:    authService,
		projectService: projectService,
		auditService:   auditService,
	}
}

func InitAuthRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	authDao := auth.NewDao(db)
	authService := auth.NewService(authDao, jwtSecret)

	auditDao := audit.NewDao(db)
	auditService := audit.NewService(auditDao)

	projectDao := project.NewDao(db)
	projectService := project.NewService(projectDao, auditService)

	authRouter := NewAuthRouter(authService, projectService, auditService)

	r.POST("/login", authRouter.Login)
	r.POST("/external/login", authRouter.ExternalLogin)
	r.POST("/register", authRouter.Register)
	r.GET("/roles", GetRoleList)
	r.GET("/status", GetStatusList)

	// Protected Verify Endpoint
	r.GET("/auth/verify", middleware.AuthMiddleware(jwtSecret), authRouter.Verify)
}

type RegisterRequest struct {
	Name       string `json:"name" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	Department string `json:"department"`
	Reason     string `json:"reason"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type ExternalLoginRequest struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required,min=6"`
	ProjectIDString string `json:"project_id_string" binding:"required"`
}

func (r *AuthRouter) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}
	user, token, err := r.authService.Login(c.Request.Context(), req.Email, req.Password)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}

	// 记录审计日志
	_ = r.auditService.LogAction(c.Request.Context(), &user.ID, constant.ActionUserLogin, "user:"+user.ID.String(), "User logged in", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data": gin.H{
			"token": token,
			"user":  user,
		},
		"code": constant.SuccessCode,
	})
}

func (r *AuthRouter) ExternalLogin(c *gin.Context) {
	var req ExternalLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}
	
	// 这里将 username 作为 email 传入 Login 方法
	user, token, err := r.authService.Login(c.Request.Context(), req.Username, req.Password)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}

	// 验证用户是否在传入的项目中
	role, err := r.projectService.VerifyUserProjectRole(c.Request.Context(), user.ID, req.ProjectIDString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}

	// 如果没有分配角色，则说明不是该项目成员
	if role == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "非该项目成员",
			"code":  constant.NotProjectMemberCode,
		})
		return
	}

	// 记录审计日志
	_ = r.auditService.LogAction(c.Request.Context(), &user.ID, constant.ActionUserExternalLogin, "project:"+req.ProjectIDString, "External login to project "+req.ProjectIDString, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data": gin.H{
			"token":           token,
			"user":            user,
			"role_in_project": role,
		},
		"code": constant.SuccessCode,
	})
}

func (r *AuthRouter) Register(c *gin.Context) {
	var req RegisterRequest
	fmt.Println("Registering user:", req)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}

	user, err := r.authService.Register(c.Request.Context(), req.Email, req.Name, req.Password, req.Department, req.Reason)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode, "message": utils.GetValidationError(err)})
		return
	}

	// 记录审计日志
	_ = r.auditService.LogAction(c.Request.Context(), &user.ID, "user_register", "user:"+user.ID.String(), "User registered, pending approval", c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "申请账号成功，等待管理员审核", "user": user, "code": constant.SuccessCode})
}

func GetRoleList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取成功", "roles": []constant.UserRole{constant.UserRoleSuperAdmin, constant.UserRoleAdmin, constant.UserRoleUser}, "code": constant.SuccessCode})
}

func GetStatusList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取成功", "status": []constant.UserStatus{constant.UserStatusActive, constant.UserStatusInactive, constant.UserStatusLocked}, "code": constant.SuccessCode})
}

func (r *AuthRouter) Verify(c *gin.Context) {
	// 1. Get User ID from Context (set by AuthMiddleware)
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "code": constant.ErrorCode})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID in token", "code": constant.ErrorCode})
		return
	}

	// 2. Get Project ID String from Query
	projectIDString := c.Query("project_id_string")
	if projectIDString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id_string is required", "code": constant.ErrorCode})
		return
	}

	// 3. Verify Role
	role, err := r.projectService.VerifyUserProjectRole(c.Request.Context(), userID, projectIDString)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"is_valid": false,
			"error":    err.Error(),
			"code":     constant.ErrorCode,
		})
		return
	}

	// 4. Return Success
	c.JSON(http.StatusOK, gin.H{
		"is_valid":        true,
		"user_id":         userID,
		"role_in_project": role,
		"code":            constant.SuccessCode,
	})
}
