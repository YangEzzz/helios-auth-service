package project

import (
	"context"
	"errors"
	"strings"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"

	"github.com/google/uuid"
)

type Service interface {
	CreateProject(ctx context.Context, opID, projectName, projectIdString, description string, roleDocumentation []string) (*models.Project, error)
	GetProjectByID(ctx context.Context, id string) (*models.Project, error)
	DeleteProject(ctx context.Context, opID, id string) error
	AddRoleTemplate(ctx context.Context, opID, projectID, roleName, description string) error
	ListRoleTemplates(ctx context.Context, projectID string) ([]models.ProjectRoleTemplate, error)
	RemoveProjectMember(ctx context.Context, opID, projectID, userID string) error
	UpdateProjectMemberRole(ctx context.Context, opID, projectID, userID, role string) error
	ListProjectMembers(ctx context.Context, projectID string) ([]models.ProjectMembership, error)
	AddProjectMember(ctx context.Context, opID, projectID, userID, role string) error
	VerifyUserProjectRole(ctx context.Context, userID uuid.UUID, projectIDString string) (string, error)
	ListProjects(ctx context.Context) ([]models.Project, error)
}

type service struct {
	dao          Dao
	auditService audit.Service
}

func NewService(dao Dao, auditService audit.Service) Service {
	return &service{dao: dao, auditService: auditService}
}

func (s *service) CreateProject(ctx context.Context, opID, projectName, projectIdString, description string, roleDocumentation []string) (*models.Project, error) {
	project := models.NewProject(projectName, projectIdString, description, roleDocumentation)
	if err := s.dao.CreateProject(project); err != nil {
		return nil, err
	}
	
	opUUID, _ := uuid.Parse(opID)
	_ = s.auditService.LogAction(ctx, &opUUID, "create_project", "project:"+project.ID.String(), "Created project "+project.ProjectName, "")
	return project, nil
}

func (s *service) AddRoleTemplate(ctx context.Context, opID, projectID, roleName, description string) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	template := models.NewProjectRoleTemplate(pID, roleName, description)
	if err := s.dao.AddRoleTemplate(template); err != nil {
		return err
	}
	opUUID, _ := uuid.Parse(opID)
	_ = s.auditService.LogAction(ctx, &opUUID, "add_role_template", "project:"+projectID, "Added role template "+roleName, "")
	return nil
}

func (s *service) ListRoleTemplates(ctx context.Context, projectID string) ([]models.ProjectRoleTemplate, error) {
	return s.dao.ListRoleTemplates(projectID)
}

func (s *service) RemoveProjectMember(ctx context.Context, opID, projectID, userID string) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	uID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	if err := s.dao.RemoveProjectMember(uID, pID); err != nil {
		return err
	}
	opUUID, _ := uuid.Parse(opID)
	_ = s.auditService.LogAction(ctx, &opUUID, "remove_member", "project:"+projectID, "Removed member "+userID, "")
	return nil
}

func (s *service) UpdateProjectMemberRole(ctx context.Context, opID, projectID, userID, role string) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	uID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	if err := s.dao.UpdateProjectMemberRole(uID, pID, role); err != nil {
		return err
	}
	opUUID, _ := uuid.Parse(opID)
	_ = s.auditService.LogAction(ctx, &opUUID, "update_member_role", "project:"+projectID, "Updated member "+userID+" role to "+role, "")
	return nil
}

func (s *service) ListProjectMembers(ctx context.Context, projectID string) ([]models.ProjectMembership, error) {
	return s.dao.ListProjectMembers(projectID)
}

func (s *service) AddProjectMember(ctx context.Context, opID, projectID, userID, role string) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	uID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	membership := models.NewProjectMembership(uID, pID, role)
	if err := s.dao.AddProjectMember(membership); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "1062") {
			return errors.New("该用户已经是本项目成员，无需重复授权")
		}
		return err
	}

	opUUID, _ := uuid.Parse(opID)
	_ = s.auditService.LogAction(ctx, &opUUID, "add_member", "project:"+projectID, "Added member "+userID+" as "+role, "")
	return nil
}

func (s *service) DeleteProject(ctx context.Context, opID, id string) error {
	if err := s.dao.DeleteProject(id); err != nil {
		return err
	}
	opUUID, _ := uuid.Parse(opID)
	_ = s.auditService.LogAction(ctx, &opUUID, "delete_project", "project:"+id, "Deleted project", "")
	return nil
}

func (s *service) GetProjectByID(ctx context.Context, id string) (*models.Project, error) {
	project, err := s.dao.GetProjectByID(id)
	if err != nil {
		if errors.Is(err, constant.ErrProjectNotFound) {
			return nil, constant.ErrProjectNotFound
		}
		return nil, err
	}
	return project, nil
}

func (s *service) VerifyUserProjectRole(ctx context.Context, userID uuid.UUID, projectIDString string) (string, error) {
	project, err := s.dao.GetProjectByProjectIDString(projectIDString)
	if err != nil {
		if errors.Is(err, constant.ErrProjectNotFound) {
			return "", constant.ErrProjectNotFound
		}
		return "", err
	}
	membership, err := s.dao.GetProjectMembership(userID, project.ID)
	if err != nil {
		if errors.Is(err, constant.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return membership.RoleInProject, nil
}

func (s *service) ListProjects(ctx context.Context) ([]models.Project, error) {
	return s.dao.ListProjects()
}
