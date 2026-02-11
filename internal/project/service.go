package project

import (
	"context"
	"helios-auth-service/internal/models"

	"github.com/google/uuid"
)

type Service interface {
	CreateProject(ctx context.Context, projectName, projectIdString, description string, roleDocumentation []string) (*models.Project, error)
	GetProjectByID(ctx context.Context, id string) (*models.Project, error)
	DeleteProject(ctx context.Context, id string) error
	AddRoleTemplate(ctx context.Context, projectID, roleName, description string) error
	VerifyUserProjectRole(ctx context.Context, userID uuid.UUID, projectIDString string) (string, error)
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

func (s *service) AddRoleTemplate(ctx context.Context, projectID, roleName, description string) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	template := models.NewProjectRoleTemplate(pID, roleName, description)
	return s.dao.AddRoleTemplate(template)
}

func (s *service) DeleteProject(ctx context.Context, id string) error {
	return s.dao.DeleteProject(id)
}

func (s *service) GetProjectByID(ctx context.Context, id string) (*models.Project, error) {
	return s.dao.GetProjectByID(id)
}

func (s *service) VerifyUserProjectRole(ctx context.Context, userID uuid.UUID, projectIDString string) (string, error) {
	// 1. Find project by string ID
	project, err := s.dao.GetProjectByProjectIDString(projectIDString)
	if err != nil {
		return "", err
	}

	// 2. Find membership
	membership, err := s.dao.GetProjectMembership(userID, project.ID)
	if err != nil {
		return "", err
	}

	return membership.RoleInProject, nil
}
