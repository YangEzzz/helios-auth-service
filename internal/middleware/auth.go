package middleware

import (
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"helios-auth-service/internal/utils"
)

// AuthMiddleware JWT 认证中间件
// 用于保护需要登录才能访问的接口
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从请求头获取 Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少身份验证信息"})
			c.Abort() // 终止请求，不再执行后续的 handler
			return
		}

		// 2. 解析 Token（格式：Bearer <token>）
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "身份验证失败"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 验证 Token
		claims, err := utils.ParseJWT(tokenString, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token缺失或无效"})
			c.Abort()
			return
		}

		// 4. 将用户 ID 存入上下文，供后续 handler 使用
		c.Set("user_id", claims.UserID.String())

		// 5. 继续执行后续的 handler
		c.Next()
	}
}

// RequireRoles limits an authenticated route to users with one of the allowed global roles.
func RequireRoles(db *gorm.DB, roles ...constant.UserRole) gin.HandlerFunc {
	allowedRoles := make(map[constant.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowedRoles[role] = struct{}{}
	}

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "code": constant.ErrorCode})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Select("id", "role").Where("id = ?", userID.(string)).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "code": constant.ErrorCode})
			c.Abort()
			return
		}

		if _, ok := allowedRoles[user.Role]; !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足", "code": constant.ErrorCode})
			c.Abort()
			return
		}

		c.Next()
	}
}
