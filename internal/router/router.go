package router

import (
	"fmt"
	"helios-auth-service/internal/router/api"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"helios-auth-service/internal/config"
)

// SetupRouter 初始化路由
func SetupRouter(cfg *config.Config) *gin.Engine {
	// 1. 初始化 GORM 数据库连接
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Successfully connected to PostgreSQL database with GORM")

	// 5. 创建 Gin 引擎
	r := gin.Default()

	// 6. 注册路由
	apiGroup := r.Group("/api")
	{
		api.InitAuthRouter(apiGroup, db, cfg.JWTSecret)
	}

	return r
}
