package api

import (
	"fmt"
	"helios-auth-service/internal/auth"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthRouter struct {
	authService auth.Service
}

// NewAuthRouter 创建 AuthRouter 实例
func NewAuthRouter(authService auth.Service) *AuthRouter {
	return &AuthRouter{
		authService: authService,
	}
}

func InitAuthRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	authDao := auth.NewDao(db)
	authService := auth.NewService(authDao, jwtSecret)
	authRouter := NewAuthRouter(authService)
	r.POST("/login", authRouter.Login)
	r.POST("/register", authRouter.Register)
	r.GET("/roles", GetRoleList)
	r.GET("/status", GetStatusList)
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (r *AuthRouter) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}
	token, err := r.authService.Login(c.Request.Context(), req.Email, req.Password)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "data": token, "code": constant.SuccessCode})
}

func (r *AuthRouter) Register(c *gin.Context) {
	var req RegisterRequest
	fmt.Println("Registering user:", req)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}

	user, err := r.authService.Register(c.Request.Context(), req.Email, req.Name, req.Password)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": utils.GetValidationError(err), "code": constant.ErrorCode})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "申请账号成功，等待管理员审核", "user": user, "code": constant.SuccessCode})
}

func GetRoleList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取成功", "roles": []constant.UserRole{constant.UserRoleSuperAdmin, constant.UserRoleAdmin, constant.UserRoleUser}, "code": constant.SuccessCode})
}

func GetStatusList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "获取成功", "status": []constant.UserStatus{constant.UserStatusActive, constant.UserStatusInactive, constant.UserStatusLocked}, "code": constant.SuccessCode})
}
