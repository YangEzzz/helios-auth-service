package main

import (
	"fmt"
	"helios-auth-service/internal/router"
	"log"
	"net/http"

	"helios-auth-service/internal/config"
)

func main() {
	// 1. 加载配置
	cfg := config.LoadConfig()

	// 5. 注册路由
	r := router.SetupRouter(cfg)

	// 8. 启动HTTP服务器
	addr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("Server starting on %s", addr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
