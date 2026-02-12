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
	GetProjectByID(id string) (*models.Project, error)
	DeleteProject(id string) error
	AddRoleTemplate(template *models.ProjectRoleTemplate) error
	GetProjectByProjectIDString(projectIDString string) (*models.Project, error)
	GetProjectMembership(userID, projectID uuid.UUID) (*models.ProjectMembership, error)
}

func NewDao(db *gorm.DB) Dao {
	return &projectDao{db: db}
}

func (p *projectDao) CreateProject(project *models.Project) error {
	return p.db.Create(project).Error
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

func (p *projectDao) AddRoleTemplate(template *models.ProjectRoleTemplate) error {
	return p.db.Create(template).Error
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
