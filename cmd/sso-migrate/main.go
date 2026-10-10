// sso-migrate applies only the additive SSO migration to the configured database.
package main

import (
	"gorm.io/gorm"
	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
	"log"
	"os"
)

func main() {
	cfg := config.LoadConfig()
	db, err := database.NewConnection(cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseName, cfg.DatabaseSSLMode)
	if err != nil {
		log.Fatal(err)
	}
	migration, err := os.ReadFile("migrations/004_sso.up.sql")
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Transaction(func(tx *gorm.DB) error { return tx.Exec(string(migration)).Error }); err != nil {
		log.Fatal(err)
	}
	log.Println("SSO migration applied; existing account and project tables unchanged")
}
