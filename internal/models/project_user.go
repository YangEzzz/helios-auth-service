package models

import (
	"time"

	"github.com/google/uuid"
)

type ProjectUser struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func NewProjectUser(projectId, userId uuid.UUID) *ProjectUser {
	return &ProjectUser{
		ProjectID: projectId,
		UserID:    userId,
		CreatedAt: time.Now(),
		ID:        uuid.New(),
	}
}
