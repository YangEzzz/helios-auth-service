package v1

import (
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/project"
	"helios-auth-service/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectRouter struct {
	projectService project.Service
}

func NewProjectRouter(projectService project.Service) *ProjectRouter {
	return &ProjectRouter{
		projectService: projectService,
	}
}

func InitProjectRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	projectDao := project.NewDao(db)
	projectService := project.NewService(projectDao)
	projectRouter := NewProjectRouter(projectService)

	projectGroup := r.Group("")
	projectGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		projectGroup.GET("/projects", projectRouter.GetProjectByID)
	}
}
