package router

import (
	"fmt"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/router/api"
	v1 "helios-auth-service/internal/router/api/v1"
	"log"
	"os"

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

	// 2. 自动迁移数据库结构（如果表不存在则新建）
	if cfg.DBAutoMigrate {
		log.Println("Initializing database tables...")
		err = db.AutoMigrate(
			&models.User{},
			&models.Project{},
			&models.ProjectMembership{},
			&models.ProjectRoleTemplate{},
			&models.AuditLog{},
		)
		if err != nil {
			log.Fatalf("Failed to auto-migrate database: %v", err)
		}
		log.Println("Database migration completed")
	} else {
		log.Println("Skipping automatic database migration (DB_AUTO_MIGRATE != true)")
	}

	// 5. 创建 Gin 引擎
	r := gin.Default()

	// 确保上传目录存在
	_ = os.MkdirAll("./uploads/avatars", os.ModePerm)

	// 提供静态文件访问
	r.Static("/uploads", "./uploads")

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
		v1.InitProjectRouter(v1Group, db, cfg.JWTSecret)
		v1.InitAuditRouter(v1Group, db, cfg.JWTSecret)
		v1.InitDashboardRouter(v1Group, db, cfg.JWTSecret)
		v1Group.POST("/upload/avatar", v1.UploadAvatar)
	}

	return r
}
