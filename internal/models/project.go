package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Project struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProjectName       string         `json:"project_name" gorm:"type:varchar(255);not null"`
	ProjectIDString   string         `json:"project_id_string" gorm:"type:varchar(255);uniqueIndex;not null"`
	ProjectURL        string         `json:"project_url" gorm:"type:varchar(1024)"`
	Description       string         `json:"description" gorm:"type:text"`
	RoleDocumentation pq.StringArray `json:"role_documentation" gorm:"type:text[]"`
	CreatedAt         time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Project) TableName() string {
	return "projects"
}

func NewProject(projectName, projectIdString, projectURL, description string, roleDocumentation []string) *Project {
	return &Project{
		ID:                uuid.New(),
		ProjectName:       projectName,
		ProjectIDString:   projectIdString,
		ProjectURL:        projectURL,
		RoleDocumentation: roleDocumentation,
		Description:       description,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}
