package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
)

func main() {
	var direction = flag.String("direction", "up", "Migration direction: up or down")
	var steps = flag.Int("steps", 0, "Number of migration steps (0 means all)")
	var version = flag.Uint("version", 0, "Migrate to specific version")
	var force = flag.Int("force", -1, "Force migration to specific version (use with caution)")
	flag.Parse()

	// 加载配置
	cfg := config.LoadConfig()

	// 连接数据库
	var db *database.DB
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
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 创建migrate实例
	driver, err := postgres.WithInstance(db.DB.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create postgres driver: %v", err)
	}

	// 获取migrations目录的绝对路径
	migrationsPath := "file://migrations"
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		// 如果当前目录没有migrations，尝试上级目录
		if _, err := os.Stat("../migrations"); err == nil {
			migrationsPath = "file://../migrations"
		} else {
			log.Fatal("Migrations directory not found. Please run from project root or ensure migrations/ directory exists.")
		}
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// 执行迁移操作
	switch {
	case *force >= 0:
		// 强制设置版本
		if err := m.Force(*force); err != nil {
			log.Fatalf("Failed to force migration to version %d: %v", *force, err)
		}
		log.Printf("Forced migration to version %d", *force)

	case *version > 0:
		// 迁移到指定版本
		if err := m.Migrate(*version); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to migrate to version %d: %v", *version, err)
		}
		log.Printf("Migrated to version %d", *version)

	case *direction == "up":
		// 向上迁移
		if *steps > 0 {
			if err := m.Steps(*steps); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("Failed to migrate up %d steps: %v", *steps, err)
			}
			log.Printf("Migrated up %d steps", *steps)
		} else {
			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("Failed to migrate up: %v", err)
			}
			log.Println("Migration up completed")
		}

	case *direction == "down":
		// 向下迁移
		if *steps > 0 {
			if err := m.Steps(-*steps); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("Failed to migrate down %d steps: %v", *steps, err)
			}
			log.Printf("Migrated down %d steps", *steps)
		} else {
			if err := m.Down(); err != nil && err != migrate.ErrNoChange {
				log.Fatalf("Failed to migrate down: %v", err)
			}
			log.Println("Migration down completed")
		}

	default:
		log.Fatalf("Invalid direction: %s. Use 'up' or 'down'", *direction)
	}

	// 显示当前版本
	version64, dirty, err := m.Version()
	if err != nil {
		log.Printf("Warning: Could not get current version: %v", err)
	} else {
		status := "clean"
		if dirty {
			status = "dirty"
		}
		log.Printf("Current database version: %d (%s)", version64, status)
	}
}
