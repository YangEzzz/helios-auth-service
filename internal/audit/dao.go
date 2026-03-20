package audit

import (
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type Dao interface {
	CreateAuditLog(log *models.AuditLog) error
	ListAuditLogs(resource string) ([]models.AuditLog, error)
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

func (d *auditDao) ListAuditLogs(resource string) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := d.db.Preload("User").Where("resource = ?", resource).Order("created_at desc").Find(&logs).Error
	return logs, err
}
