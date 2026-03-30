package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID    *uuid.UUID `json:"user_id" gorm:"type:uuid;index"` // 指针类型允许为NULL
	Action    string     `json:"action" gorm:"type:varchar(100);not null;index"`
	Resource  string     `json:"resource" gorm:"type:varchar(255)"`
	Details   string     `json:"details" gorm:"type:text"`
	IPAddress string     `json:"ip_address" gorm:"type:varchar(45)"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime;index"`

	// Virtual fields for translated names
	ActionName   string `json:"action_name" gorm:"-"`
	ResourceName string `json:"resource_name" gorm:"-"`
	DetailsName  string `json:"details_name" gorm:"-"`

	// Associations
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

func NewAuditLog(userID *uuid.UUID, action, resource, details, ipAddress string) *AuditLog {
	return &AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}
}
