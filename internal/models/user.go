package models

import (
	"helios-auth-service/internal/constant"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Username     string              `json:"username" gorm:"type:varchar(255)"`
	Email        string              `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string              `json:"-" gorm:"type:varchar(255);not null"` // 密码哈希不应暴露给前端
	Role         constant.UserRole   `json:"role" gorm:"type:varchar(50)"`
	CreatedAt    time.Time           `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
	Status       constant.UserStatus `json:"status" gorm:"type:varchar(50)"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

func NewUser(username, email, passwordHash string) *User {
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         constant.UserRoleUser,
		CreatedAt:    time.Now(),
		Status:       constant.UserStatusActive,
	}
}
