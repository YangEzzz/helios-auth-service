package project

import (
	"context"
	"helios-auth-service/internal/models"
)

type Service interface {
	CreateProject(ctx context.Context, projectName, projectIdString, description string, roleDocumentation []string) (*models.Project, error)
	GetProjectByID(ctx context.Context, id string) (*models.Project, error)
}

type service struct {
	dao Dao
}

func NewService(dao Dao) Service {
	return &service{dao: dao}
}

func (s *service) CreateProject(ctx context.Context, projectName, projectIdString, description string, roleDocumentation []string) (*models.Project, error) {
	project := models.NewProject(projectName, projectIdString, description, roleDocumentation)
	err := s.dao.CreateProject(project)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (s *service) GetProjectByID(ctx context.Context, id string) (*models.Project, error) {
	return s.dao.GetProjectByID(id)
}
