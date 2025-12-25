package project

import (
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type projectDao struct {
	db *gorm.DB
}

type Dao interface {
	CreateProject(project *models.Project) error
	GetProjectByID(id string) (*models.Project, error)
}

func NewDao(db *gorm.DB) Dao {
	return &projectDao{db: db}
}

func (p *projectDao) CreateProject(project *models.Project) error {
	return p.db.Create(project).Error
}

func (p *projectDao) GetProjectByID(id string) (*models.Project, error) {
	var project models.Project
	err := p.db.Where("id = ?", id).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}
