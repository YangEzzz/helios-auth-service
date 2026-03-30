package dashboard

import (
	"time"

	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Dao interface {
	GetUserByID(id string) (*models.User, error)
	CountUsers() (int64, error)
	CountUsersByStatus(status constant.UserStatus) (int64, error)
	CountProjects() (int64, error)
	CountAuditLogs() (int64, error)
	CountUsersCreatedAfter(start time.Time) (int64, error)
	CountProjectsCreatedAfter(start time.Time) (int64, error)
	CountAuditLogsCreatedAfter(start time.Time) (int64, error)
	ListRecentAuditLogs(limit int) ([]models.AuditLog, error)
	ListRecentAuditLogsForUser(userID uuid.UUID, limit int) ([]models.AuditLog, error)
	ListRecentDecisionLogs(limit int) ([]models.AuditLog, error)
	ListProjectMembershipsForUser(userID uuid.UUID) ([]models.ProjectMembership, error)
	GetLastLoginLogForUser(userID uuid.UUID) (*models.AuditLog, error)
	ListProjectCreationTrend(start time.Time) ([]DailyCount, error)
}

type DailyCount struct {
	Day   time.Time `json:"day"`
	Count int64     `json:"count"`
}

type dao struct {
	db *gorm.DB
}

func NewDao(db *gorm.DB) Dao {
	return &dao{db: db}
}

func (d *dao) GetUserByID(id string) (*models.User, error) {
	var user models.User
	if err := d.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *dao) CountUsers() (int64, error) {
	var count int64
	err := d.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (d *dao) CountUsersByStatus(status constant.UserStatus) (int64, error) {
	var count int64
	err := d.db.Model(&models.User{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (d *dao) CountProjects() (int64, error) {
	var count int64
	err := d.db.Model(&models.Project{}).Count(&count).Error
	return count, err
}

func (d *dao) CountAuditLogs() (int64, error) {
	var count int64
	err := d.db.Model(&models.AuditLog{}).Count(&count).Error
	return count, err
}

func (d *dao) CountUsersCreatedAfter(start time.Time) (int64, error) {
	var count int64
	err := d.db.Model(&models.User{}).Where("created_at >= ?", start).Count(&count).Error
	return count, err
}

func (d *dao) CountProjectsCreatedAfter(start time.Time) (int64, error) {
	var count int64
	err := d.db.Model(&models.Project{}).Where("created_at >= ?", start).Count(&count).Error
	return count, err
}

func (d *dao) CountAuditLogsCreatedAfter(start time.Time) (int64, error) {
	var count int64
	err := d.db.Model(&models.AuditLog{}).Where("created_at >= ?", start).Count(&count).Error
	return count, err
}

func (d *dao) ListRecentAuditLogs(limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := d.db.Preload("User").Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

func (d *dao) ListRecentAuditLogsForUser(userID uuid.UUID, limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := d.db.Preload("User").
		Where("user_id = ? OR resource = ?", userID, "user:"+userID.String()).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (d *dao) ListRecentDecisionLogs(limit int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := d.db.Preload("User").
		Where("action IN ?", []string{constant.ActionApproveUser, constant.ActionRejectUser}).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (d *dao) ListProjectMembershipsForUser(userID uuid.UUID) ([]models.ProjectMembership, error) {
	var memberships []models.ProjectMembership
	err := d.db.Preload("Project").Where("user_id = ?", userID).Order("created_at desc").Find(&memberships).Error
	return memberships, err
}

func (d *dao) GetLastLoginLogForUser(userID uuid.UUID) (*models.AuditLog, error) {
	var log models.AuditLog
	err := d.db.Where("user_id = ? AND action IN ?", userID, []string{constant.ActionUserLogin, constant.ActionUserExternalLogin}).
		Order("created_at desc").
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (d *dao) ListProjectCreationTrend(start time.Time) ([]DailyCount, error) {
	var result []DailyCount
	err := d.db.Model(&models.Project{}).
		Select("DATE(created_at) as day, COUNT(*) as count").
		Where("created_at >= ?", start).
		Group("DATE(created_at)").
		Order("day asc").
		Scan(&result).Error
	return result, err
}
