package api

import (
	"helios-auth-service/internal/auth"
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
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (r *AuthRouter) Login(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := r.authService.Login(c.Request.Context(), req.Email, req.Password)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Login successful", "user": user})
}

func (r *AuthRouter) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := r.authService.Register(c.Request.Context(), req.Email, req.Name, req.Password)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Registration successful", "user": user})
}
