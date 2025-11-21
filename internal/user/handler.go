package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler 用户处理器
type Handler struct {
	userService UserService
}

// NewHandler 创建用户处理器实例
func NewHandler(userService UserService) *Handler {
	return &Handler{
		userService: userService,
	}
}

// RegisterRequest 注册请求结构
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterResponse 注册响应结构
type RegisterResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
	UserID  string `json:"user_id"`
}

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 注册用户
	user, err := h.userService.RegisterUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// 根据错误类型返回不同的状态码
		if err.Error() == "email already registered" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动登录，生成 Token
	token, err := h.userService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration successful, but failed to login automatically"})
		return
	}

	c.JSON(http.StatusOK, RegisterResponse{
		Message: "Registration successful",
		Token:   token,
		UserID:  user.ID.String(),
	})
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.userService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Message: "Login successful",
		Token:   token,
	})
}

// GetProfile 获取用户个人信息（需要认证）
func (h *Handler) GetProfile(c *gin.Context) {
	// 从上下文中获取用户 ID（由认证中间件设置）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
