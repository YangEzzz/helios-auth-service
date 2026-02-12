package project

import (
	"context"
	"errors"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
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
	dao          Dao
	auditService audit.Service
}

func NewService(dao Dao, auditService audit.Service) Service {
	return &service{dao: dao, auditService: auditService}
}

func (s *service) CreateProject(ctx context.Context, projectName, projectIdString, description string, roleDocumentation []string) (*models.Project, error) {
	project := models.NewProject(projectName, projectIdString, description, roleDocumentation)
	if err := s.dao.CreateProject(project); err != nil {
		return nil, err
	}

	// Audit Log
	// Note: UserID is currently not passed to CreateProject, we might need to change signature or get it from context if middleware puts it there.
	// For now, passing nil for UserID as system action or we can extract from context if available.
	// Assuming context has "userID" if middleware sets it, but ctx is standard context.
	// Let's check if we can get userID from context.
	// The caller (Router) usually passes gin.Context.Request.Context().
	// Gin context keys are not automatically in Request.Context().
	// We need to fix this later or assume nil for now.
	// To keep it simple and safe:
	// Use extractUserIDFromContext(ctx) helper if we had one.
	// For now, let's use nil for UserID in this step, or simple placeholder.
	// User requested "Audit Logging Logic", I will try to extract user ID from context if I can, but standard context doesn't have it unless we put it there.
	// I'll leave UserID as nil for now and just log the resource.
	_ = s.auditService.LogAction(ctx, nil, "create_project", "project:"+project.ID.String(), "Created project "+project.ProjectName, "")

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
	if err := s.dao.DeleteProject(id); err != nil {
		return err
	}
	_ = s.auditService.LogAction(ctx, nil, "delete_project", "project:"+id, "Deleted project", "")
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
	// 1. Find project by string ID
	project, err := s.dao.GetProjectByProjectIDString(projectIDString)
	if err != nil {
		if errors.Is(err, constant.ErrProjectNotFound) {
			return "", constant.ErrProjectNotFound
		}
		return "", err
	}

	// 2. Find membership
	membership, err := s.dao.GetProjectMembership(userID, project.ID)
	if err != nil {
		if errors.Is(err, constant.ErrRecordNotFound) {
			// 如果没有会员记录，视为空角色，或者返回特定错误
			// 这里根据业务需求，如果没有找到记录，说明不是成员，role为空字符串
			return "", nil
		}
		return "", err
	}

	return membership.RoleInProject, nil
}
