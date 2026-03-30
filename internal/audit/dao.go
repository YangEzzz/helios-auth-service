package audit

import (
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type Dao interface {
	CreateAuditLog(log *models.AuditLog) error
	ListAuditLogs(resource string, page, pageSize int) ([]models.AuditLog, int64, error)
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

func (d *auditDao) ListAuditLogs(resource string, page, pageSize int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64
	query := d.db.Model(&models.AuditLog{})
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = query.Preload("User").Order("created_at desc").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}
