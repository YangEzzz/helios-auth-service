package models

import (
	"time"

	"github.com/google/uuid"
)

type ProjectRoleTemplate struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProjectID   uuid.UUID `json:"project_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_project_role_name"`
	RoleName    string    `json:"role_name" gorm:"type:varchar(50);not null;uniqueIndex:idx_project_role_name"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (ProjectRoleTemplate) TableName() string {
	return "project_role_templates"
}

func NewProjectRoleTemplate(projectID uuid.UUID, roleName, description string) *ProjectRoleTemplate {
	return &ProjectRoleTemplate{
		ID:          uuid.New(),
		ProjectID:   projectID,
		RoleName:    roleName,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
