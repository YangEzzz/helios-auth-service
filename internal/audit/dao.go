package audit

import (
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type Dao interface {
	CreateAuditLog(log *models.AuditLog) error
}

type auditDao struct {
	db *gorm.DB
}

func NewDao(db *gorm.DB) Dao {
	return &auditDao{db: db}
}

func (d *auditDao) CreateAuditLog(log *models.AuditLog) error {
	return d.db.Create(log).Error
}
