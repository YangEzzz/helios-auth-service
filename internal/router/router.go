package router

import (
	"fmt"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/router/api"
	v1 "helios-auth-service/internal/router/api/v1"
	"log"

	"helios-auth-service/internal/config"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化路由
func SetupRouter(cfg *config.Config) *gin.Engine {
	// 1. 初始化 GORM 数据库连接
	fmt.Printf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)

	db, err := database.NewConnection(
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Successfully connected to PostgreSQL database with GORM")

	// 5. 创建 Gin 引擎
	r := gin.Default()

	// CORS设置 (示例)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 6. 注册路由
	apiGroup := r.Group("/api")
	{
		api.InitAuthRouter(apiGroup, db, cfg.JWTSecret)
	}
	v1Group := r.Group("/api/v1")
	{
		v1.InitUserRouter(v1Group, db, cfg.JWTSecret)
	}

	return r
}
