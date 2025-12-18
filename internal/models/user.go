package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole 用户角色类型
type UserRole string

const (
	UserRoleSuperAdmin UserRole = "super_admin"
	UserRoleAdmin      UserRole = "admin"
	UserRoleUser       UserRole = "user"
)

// UserStatus 用户状态类型
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

type User struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Username     string     `json:"username" gorm:"type:varchar(255)"`
	Email        string     `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string     `json:"-" gorm:"type:varchar(255);not null"` // 密码哈希不应暴露给前端
	Role         UserRole   `json:"role" gorm:"type:varchar(50)"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	Status       UserStatus `json:"status" gorm:"type:varchar(50)"`
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
		Role:         UserRoleUser,
		CreatedAt:    time.Now(),
		Status:       UserStatusInactive,
	}
}
