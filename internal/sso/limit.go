package sso

import (
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

// LoginLimit bounds password attempts per source. Multi-instance deployments
// should additionally use their ingress rate limiter.
func LoginLimit() gin.HandlerFunc {
	type bucket struct {
		Count   int
		Expires time.Time
	}
	var mu sync.Mutex
	attempts := map[string]bucket{}
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		mu.Lock()
		if len(attempts) > 10000 {
			for ip, b := range attempts {
				if b.Expires.Before(now) {
					delete(attempts, ip)
				}
			}
		}
		b := attempts[key]
		if b.Expires.Before(now) {
			b = bucket{Expires: now.Add(time.Minute)}
		}
		b.Count++
		attempts[key] = b
		denied := b.Count > 10
		mu.Unlock()
		if denied {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(429, gin.H{"code": 429, "message": "登录尝试过于频繁，请一分钟后重试"})
			return
		}
		c.Next()
	}
}
