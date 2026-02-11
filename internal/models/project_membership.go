package models

import (
	"time"

	"github.com/google/uuid"
)

type ProjectMembership struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID        uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_user_project"`
	ProjectID     uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_user_project"`
	RoleInProject string    `json:"role_in_project" gorm:"type:varchar(50);not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Preload associations
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Project *Project `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
}

// TableName 指定表名
func (ProjectMembership) TableName() string {
	return "project_memberships"
}

func NewProjectMembership(userID, projectID uuid.UUID, roleInProject string) *ProjectMembership {
	return &ProjectMembership{
		ID:            uuid.New(),
		UserID:        userID,
		ProjectID:     projectID,
		RoleInProject: roleInProject,
		CreatedAt:     time.Now(),
	}
}
