package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID              uuid.UUID `json:"id"`
	ProjectName     string    `json:"project_name"`
	ProjectIDString string    `json:"project_id_string"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewProject(projectName, projectIdString, description string) *Project {
	return &Project{
		ID:              uuid.New(),
		ProjectName:     projectName,
		ProjectIDString: projectIdString,
		Description:     description,
		CreatedAt:       time.Now(),
	}
}
