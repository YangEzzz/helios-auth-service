package sso

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"helios-auth-service/internal/middleware"
)

var errImageCredentials = errors.New("invalid image client credentials")

// ImageAuth allows a project backend to act for an active project member.
// Project credentials alone never authorize a request; the asserted user must
// still have project membership. Cookie/Bearer callers retain existing checks.
func ImageAuth(db *gorm.DB, jwtSecret string) gin.HandlerFunc {
	h := &Handler{DB: db}
	return imageAuth(middleware.AuthMiddleware(jwtSecret, db), func(c *gin.Context, userID uuid.UUID) error {
		client, err := h.authenticate(c)
		if err != nil {
			return errImageCredentials
		}
		_, err = identity(db.WithContext(c.Request.Context()), client, userID)
		return err
	})
}

func imageAuth(userAuth gin.HandlerFunc, verify func(*gin.Context, uuid.UUID) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.GetHeader("Authorization"), "Basic ") {
			userAuth(c)
			return
		}
		userID, err := uuid.Parse(c.GetHeader("X-Helios-User-ID"))
		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"code": 400, "message": "请提供有效的 Helios 用户 ID"})
			return
		}
		if err = verify(c, userID); err != nil {
			status, message := 503, "暂时无法校验图片服务权限"
			if errors.Is(err, errImageCredentials) {
				status, message = 401, "项目凭据无效"
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status, message = 403, "账号或项目权限已失效"
			}
			c.AbortWithStatusJSON(status, gin.H{"code": status, "message": message})
			return
		}
		c.Set("user_id", userID.String())
		c.Next()
	}
}
