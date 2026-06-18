package project

import (
	"errors"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type projectDao struct {
	db *gorm.DB
}

type Dao interface {
	CreateProject(project *models.Project) error
	UpdateProject(id uuid.UUID, projectName, projectURL, description string) (*models.Project, error)
	GetProjectByID(id string) (*models.Project, error)
	DeleteProject(id string) error
	AddRoleTemplate(template *models.ProjectRoleTemplate) error
	ListRoleTemplates(projectID string) ([]models.ProjectRoleTemplate, error)
	RoleTemplateExists(projectID uuid.UUID, roleName string) (bool, error)
	GetProjectByProjectIDString(projectIDString string) (*models.Project, error)
	GetProjectMembership(userID, projectID uuid.UUID) (*models.ProjectMembership, error)
	RemoveProjectMember(userID, projectID uuid.UUID) error
	UpdateProjectMemberRole(userID, projectID uuid.UUID, role string) error
	ListProjectMembers(projectID string) ([]models.ProjectMembership, error)
	AddProjectMember(membership *models.ProjectMembership) error
	ListProjects() ([]models.Project, error)
	ListProjectsForUser(userID uuid.UUID) ([]models.ProjectMembership, error)
}

func NewDao(db *gorm.DB) Dao {
	return &projectDao{db: db}
}

func (p *projectDao) CreateProject(project *models.Project) error {
	return p.db.Create(project).Error
}

func (p *projectDao) UpdateProject(id uuid.UUID, projectName, projectURL, description string) (*models.Project, error) {
	var project models.Project
	if err := p.db.Where("id = ?", id).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrProjectNotFound
		}
		return nil, err
	}

	project.ProjectName = projectName
	project.ProjectURL = projectURL
	project.Description = description
	if err := p.db.Save(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (p *projectDao) GetProjectByProjectIDString(projectIDString string) (*models.Project, error) {
	var project models.Project
	err := p.db.Where("project_id_string = ?", projectIDString).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

func (p *projectDao) GetProjectMembership(userID, projectID uuid.UUID) (*models.ProjectMembership, error) {
	var membership models.ProjectMembership
	err := p.db.Where("user_id = ? AND project_id = ?", userID, projectID).First(&membership).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrRecordNotFound
		}
		return nil, err
	}
	return &membership, nil
}

func (p *projectDao) RemoveProjectMember(userID, projectID uuid.UUID) error {
	return p.db.Where("user_id = ? AND project_id = ?", userID, projectID).Delete(&models.ProjectMembership{}).Error
}

func (p *projectDao) UpdateProjectMemberRole(userID, projectID uuid.UUID, role string) error {
	return p.db.Model(&models.ProjectMembership{}).Where("user_id = ? AND project_id = ?", userID, projectID).Update("role_in_project", role).Error
}

func (p *projectDao) ListProjectMembers(projectID string) ([]models.ProjectMembership, error) {
	var members []models.ProjectMembership
	err := p.db.Preload("User").Where("project_id = ?", projectID).Find(&members).Error
	return members, err
}

func (p *projectDao) AddProjectMember(membership *models.ProjectMembership) error {
	return p.db.Create(membership).Error
}

func (p *projectDao) AddRoleTemplate(template *models.ProjectRoleTemplate) error {
	return p.db.Create(template).Error
}

func (p *projectDao) ListRoleTemplates(projectID string) ([]models.ProjectRoleTemplate, error) {
	var templates []models.ProjectRoleTemplate
	err := p.db.Where("project_id = ?", projectID).Find(&templates).Error
	return templates, err
}

func (p *projectDao) RoleTemplateExists(projectID uuid.UUID, roleName string) (bool, error) {
	var count int64
	err := p.db.Model(&models.ProjectRoleTemplate{}).
		Where("project_id = ? AND role_name = ?", projectID, roleName).
		Count(&count).Error
	return count > 0, err
}

func (p *projectDao) DeleteProject(id string) error {
	return p.db.Where("id = ?", id).Delete(&models.Project{}).Error
}

func (p *projectDao) GetProjectByID(id string) (*models.Project, error) {
	var project models.Project
	err := p.db.Where("id = ?", id).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}
func (p *projectDao) ListProjects() ([]models.Project, error) {
	var projects []models.Project
	err := p.db.Find(&projects).Error
	return projects, err
}

func (p *projectDao) ListProjectsForUser(userID uuid.UUID) ([]models.ProjectMembership, error) {
	var memberships []models.ProjectMembership
	err := p.db.Preload("Project").Where("user_id = ?", userID).Order("created_at desc").Find(&memberships).Error
	return memberships, err
}
