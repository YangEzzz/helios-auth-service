package v1

import (
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/project"

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

func (r *ProjectRouter) GetProjectByID(c *gin.Context) {
	id := c.Query("id")
	projects, err := r.projectService.GetProjectByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"project": projects, "code": constant.SuccessCode})
}
