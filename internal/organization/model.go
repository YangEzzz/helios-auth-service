package organization

import (
	"github.com/google/uuid"
	"time"
)

type Department struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	ParentID     *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	SortOrder    int        `json:"sort_order"`
	Status       string     `json:"status"`
	Version      int        `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MemberCount  int64      `json:"member_count" gorm:"->;-:migration"`
	ManagerCount int64      `json:"manager_count" gorm:"->;-:migration"`
}

type Membership struct {
	DepartmentID uuid.UUID `json:"department_id" gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey"`
	IsManager    bool      `json:"is_manager"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Membership) TableName() string { return "department_memberships" }

type Member struct {
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	IsManager bool      `json:"is_manager"`
}
