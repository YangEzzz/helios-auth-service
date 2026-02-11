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
		projectGroup.POST("/projects", projectRouter.CreateProject)
		projectGroup.POST("/delete_project", projectRouter.DeleteProject)
		projectGroup.POST("/add_project_template", projectRouter.AddRoleTemplate)
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

type CreateProjectRequest struct {
	ProjectName     string `json:"project_name" binding:"required"`
	ProjectIDString string `json:"project_id_string" binding:"required"`
	Description     string `json:"description"`
}

func (r *ProjectRouter) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}

	// roleDocumentation is deprecated in favor of project_role_templates, passing empty slice
	project, err := r.projectService.CreateProject(c.Request.Context(), req.ProjectName, req.ProjectIDString, req.Description, []string{})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}

	c.JSON(200, gin.H{"message": "Project created successfully", "project": project, "code": constant.SuccessCode})
}

type DeleteProjectRequest struct {
	ID string `json:"id" binding:"required"`
}

func (r *ProjectRouter) DeleteProject(c *gin.Context) {
	var req DeleteProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	if err := r.projectService.DeleteProject(c.Request.Context(), req.ID); err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "Project deleted successfully", "code": constant.SuccessCode})
}

type AddRoleTemplateRequest struct {
	ProjectID   string `json:"project_id" binding:"required"`
	RoleName    string `json:"role_name" binding:"required"`
	Description string `json:"description"`
}

func (r *ProjectRouter) AddRoleTemplate(c *gin.Context) {
	var req AddRoleTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	if err := r.projectService.AddRoleTemplate(c.Request.Context(), req.ProjectID, req.RoleName, req.Description); err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "Role template added successfully", "code": constant.SuccessCode})
}
