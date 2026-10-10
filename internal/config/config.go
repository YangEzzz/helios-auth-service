package config

import (
	"log"
	"net/url"
	"os"
	"strings"

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
	FrontendURL      string
	CookieSecure     bool
	TrustedProxies   []string
}

func LoadConfig() *Config {
	switch {
	case godotenv.Load(".env") == nil:
		log.Println("Loaded configuration from .env")
	case godotenv.Load(".env-production") == nil:
		log.Println("Loaded configuration from .env-production")
	default:
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
		FrontendURL:      getEnv("FRONTEND_URL", "http://localhost:5174"),
		CookieSecure:     getEnv("COOKIE_SECURE", "true") == "true",
		TrustedProxies:   strings.Split(getEnv("TRUSTED_PROXIES", "127.0.0.1,::1"), ","),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("Missing critical environment variables (JWT_SECRET)")
	}
	u, err := url.Parse(cfg.FrontendURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
		log.Fatal("FRONTEND_URL must use HTTPS, or HTTP on loopback for local development")
	}
	if !cfg.CookieSecure && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		log.Fatal("COOKIE_SECURE=false is only allowed for loopback development")
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
