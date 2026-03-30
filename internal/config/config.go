package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	JWTSecret string
	// 数据库配置
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string
	DBAutoMigrate    bool
}

func LoadConfig() *Config {
	if err := godotenv.Load(".env-production"); err != nil {
		log.Println("No .env file found, loading from environment variables")
	}

	cfg := &Config{
		Port:      getEnv("PORT", "3456"),
		JWTSecret: getEnv("JWT_SECRET", "supersecretjwtkey"),
		// 数据库配置
		DatabaseHost:     getEnv("DB_HOST", "localhost"),
		DatabasePort:     getEnv("DB_PORT", "5432"),
		DatabaseUser:     getEnv("DB_USER", "postgres"),
		DatabasePassword: getEnv("DB_PASSWORD", ""),
		DatabaseName:     getEnv("DB_NAME", "system_db"),
		DatabaseSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBAutoMigrate:    getEnv("DB_AUTO_MIGRATE", "true") == "true",
	}

	if cfg.JWTSecret == "" {
		log.Fatal("Missing critical environment variables (JWT_SECRET)")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
