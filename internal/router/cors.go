package router

import (
	"github.com/gin-gonic/gin"
	"helios-auth-service/internal/session"
)

// Nginx proxies API requests and preflights; the service owns browser CORS.
func browserCORS(frontend string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := session.Origin(frontend)
		c.Header("Vary", "Origin")
		if c.GetHeader("Origin") == origin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			if _, err := c.Cookie(session.CookieName); err == nil && !session.Guard(c, origin) {
				return
			}
		}
		c.Next()
	}
}
