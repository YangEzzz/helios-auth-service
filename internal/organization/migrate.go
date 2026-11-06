package organization

import (
	_ "embed"
	"gorm.io/gorm"
)

//go:embed schema.sql
var schema string

// Migrate is explicit and additive. It never imports the legacy free-text department field.
func Migrate(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(72410831)").Error; err != nil {
			return err
		}
		return tx.Exec(schema).Error
	})
}
