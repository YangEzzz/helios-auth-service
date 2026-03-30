package v1

import (
	"strconv"

	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/project"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectRouter struct {
	projectService project.Service
	auditService   audit.Service
}

func NewProjectRouter(projectService project.Service, auditService audit.Service) *ProjectRouter {
	return &ProjectRouter{
		projectService: projectService,
		auditService:   auditService,
	}
}

func InitProjectRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	auditDao := audit.NewDao(db)
	auditService := audit.NewService(auditDao)

	projectDao := project.NewDao(db)
	projectService := project.NewService(projectDao, auditService)
	projectRouter := NewProjectRouter(projectService, auditService)

	projectGroup := r.Group("")
	projectGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		projectGroup.GET("/projects", projectRouter.ListProjects)
		projectGroup.GET("/my-projects", projectRouter.ListMyProjects)
		projectGroup.GET("/project", projectRouter.GetProjectByID)
		projectGroup.GET("/project/role_templates", projectRouter.ListRoleTemplates)
		projectGroup.GET("/project/members", projectRouter.ListProjectMembers)
		projectGroup.GET("/project/audit_logs", projectRouter.ListAuditLogs)

		projectGroup.POST("/projects", projectRouter.CreateProject)
		projectGroup.POST("/delete_project", projectRouter.DeleteProject)
		projectGroup.POST("/add_project_template", projectRouter.AddRoleTemplate)

		// 核心成员管理
		projectGroup.POST("/project/member", projectRouter.AddMember)
		projectGroup.POST("/project/member/remove", projectRouter.RemoveMember)
		projectGroup.POST("/project/member/role", projectRouter.UpdateMemberRole)
	}
}

func (r *ProjectRouter) ListProjects(c *gin.Context) {
	projects, err := r.projectService.ListProjects(c.Request.Context())
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"projects": projects, "code": constant.SuccessCode})
}

func (r *ProjectRouter) ListMyProjects(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"message": "Unauthorized", "code": constant.ErrorCode})
		return
	}

	projects, err := r.projectService.ListMyProjects(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	c.JSON(200, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"projects": projects,
		},
		"code": constant.SuccessCode,
	})
}

func (r *ProjectRouter) GetProjectByID(c *gin.Context) {
	id := c.Query("id")
	projects, err := r.projectService.GetProjectByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"project": projects, "code": constant.SuccessCode})
}

func (r *ProjectRouter) ListRoleTemplates(c *gin.Context) {
	id := c.Query("id")
	templates, err := r.projectService.ListRoleTemplates(c.Request.Context(), id)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"role_templates": templates, "code": constant.SuccessCode})
}

func (r *ProjectRouter) ListAuditLogs(c *gin.Context) {
	projectId := c.Query("id")
	resource := "project:" + projectId

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	logs, total, err := r.auditService.ListAuditLogs(c.Request.Context(), resource, page, pageSize)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
		"code": constant.SuccessCode,
	})
}

// ---------------- 成员管理处理函数 ----------------

type AddMemberRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Role      string `json:"role" binding:"required"`
}

func (r *ProjectRouter) AddMember(c *gin.Context) {
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	opID, _ := c.Get("user_id")
	if err := r.projectService.AddProjectMember(c.Request.Context(), opID.(string), req.ProjectID, req.UserID, req.Role); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "Member added successfully", "code": constant.SuccessCode})
}

func (r *ProjectRouter) ListProjectMembers(c *gin.Context) {
	id := c.Query("project_id")
	members, err := r.projectService.ListProjectMembers(c.Request.Context(), id)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"members": members, "code": constant.SuccessCode})
}

func (r *ProjectRouter) RemoveMember(c *gin.Context) {
	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		UserID    string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	opID, _ := c.Get("user_id")
	if err := r.projectService.RemoveProjectMember(c.Request.Context(), opID.(string), req.ProjectID, req.UserID); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "Member removed successfully", "code": constant.SuccessCode})
}

func (r *ProjectRouter) UpdateMemberRole(c *gin.Context) {
	var req struct {
		ProjectID string `json:"project_id" binding:"required"`
		UserID    string `json:"user_id" binding:"required"`
		Role      string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	opID, _ := c.Get("user_id")
	if err := r.projectService.UpdateProjectMemberRole(c.Request.Context(), opID.(string), req.ProjectID, req.UserID, req.Role); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "Role updated successfully", "code": constant.SuccessCode})
}

// ---------------- 基础项目指令 ----------------

type CreateProjectRequest struct {
	ProjectName     string `json:"project_name" binding:"required"`
	ProjectIDString string `json:"project_id_string" binding:"required"`
	Description     string `json:"description"`
}

func (r *ProjectRouter) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	opID, _ := c.Get("user_id")
	project, err := r.projectService.CreateProject(c.Request.Context(), opID.(string), req.ProjectName, req.ProjectIDString, req.Description, []string{})
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
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
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	opID, _ := c.Get("user_id")
	if err := r.projectService.DeleteProject(c.Request.Context(), opID.(string), req.ID); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
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
		c.JSON(400, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	opID, _ := c.Get("user_id")
	if err := r.projectService.AddRoleTemplate(c.Request.Context(), opID.(string), req.ProjectID, req.RoleName, req.Description); err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"message": "Role template added successfully", "code": constant.SuccessCode})
}
