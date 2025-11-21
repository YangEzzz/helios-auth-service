package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/repository"
	"helios-auth-service/internal/user"
)

func main() {
	// 1. 加载配置
	cfg := config.LoadConfig()

	// 2. 初始化数据库连接和仓库层
	var userRepo repository.UserRepository
	var db *database.DB

	// 检查是否配置了数据库连接
	if cfg.DatabaseHost != "" && cfg.DatabasePassword != "" {
		// 使用分离的数据库配置参数连接
		var err error
		db, err = database.NewConnection(
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseUser,
			cfg.DatabasePassword,
			cfg.DatabaseName,
			cfg.DatabaseSSLMode,
		)
		if err != nil {
			log.Printf("Failed to connect to database: %v", err)
			log.Println("Falling back to in-memory storage")
			userRepo = repository.NewMockUserRepository()
		} else {
			userRepo = repository.NewPostgresUserRepository(db.DB)
			log.Println("Using PostgreSQL database")
		}
	} else {
		// 使用内存存储
		userRepo = repository.NewMockUserRepository()
		log.Println("Using in-memory storage (no database configured)")
	}

	// 确保在程序退出时关闭数据库连接
	if db != nil {
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}()
	}

	// 3. 初始化服务层
	userService := user.NewUserService(userRepo, cfg.JWTSecret)

	// 4. 初始化HTTP路由器
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

	// 5. 初始化处理器
	userHandler := user.NewHandler(userService)

	// 6. 注册公开路由（不需要认证）
	publicRoutes := r.Group("/api/v1")
	{
		publicRoutes.POST("/register", userHandler.Register)
		publicRoutes.POST("/login", userHandler.Login)
	}

	// 7. 注册受保护路由（需要认证）
	protectedRoutes := r.Group("/api/v1")
	protectedRoutes.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		protectedRoutes.GET("/profile", userHandler.GetProfile)
		// 可以在这里添加更多需要认证的路由
	}

	// 8. 启动HTTP服务器
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	log.Println("Public routes:")
	log.Println("  POST /api/v1/register - 用户注册")
	log.Println("  POST /api/v1/login - 用户登录")
	log.Println("Protected routes (需要 JWT Token):")
	log.Println("  GET /api/v1/profile - 获取用户信息")
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
