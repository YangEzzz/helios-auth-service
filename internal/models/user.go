package models

import (
	"helios-auth-service/internal/constant"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Username     string              `json:"name" gorm:"type:varchar(255)"`
	Email        string              `json:"email" gorm:"type:varchar(255);unique;not null"`
	PasswordHash string              `json:"-" gorm:"type:varchar(255);not null"` // 密码哈希不应暴露给前端
	Role         constant.UserRole   `json:"role" gorm:"type:varchar(50)"`
	CreatedAt    time.Time           `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
	Status       constant.UserStatus `json:"status" gorm:"type:varchar(50)"`
	Department   string              `json:"department" gorm:"type:varchar(255)"`
	Reason       string              `json:"reason" gorm:"type:text"`
	Avatar       string              `json:"avatar" gorm:"type:varchar(511);default:''"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

func NewUser(username, email, passwordHash, department, reason, avatar string) *User {
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         constant.UserRoleUser,
		CreatedAt:    time.Now(),
		Status:       constant.UserStatusPending, // 注册后默认为待处理
		Department:   department,
		Reason:       reason,
		Avatar:       avatar,
	}
}
