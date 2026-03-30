package v1

import (
	"strconv"
	"strings"

	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


type AuditRouter struct {
	auditService audit.Service
	db           *gorm.DB
}

func NewAuditRouter(auditService audit.Service, db *gorm.DB) *AuditRouter {
	return &AuditRouter{
		auditService: auditService,
		db:           db,
	}
}

func InitAuditRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	auditDao := audit.NewDao(db)
	auditService := audit.NewService(auditDao)
	auditRouter := NewAuditRouter(auditService, db)

	auditGroup := r.Group("")
	auditGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		// 也许需要一个 middleware 来限制只有 admin 可以访问
		// 目前先加上登录验证
		auditGroup.GET("/audit_logs", auditRouter.ListAuditLogs)
	}
}

func (r *AuditRouter) ListAuditLogs(c *gin.Context) {
	resource := c.Query("resource") // optional
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := r.auditService.ListAuditLogs(c.Request.Context(), resource, page, pageSize)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	// 尝试反查资源名称（如项目名、用户名）使前端展示更友好
	for i := range logs {
		if strings.HasPrefix(logs[i].Resource, "project:") {
			pID := strings.TrimPrefix(logs[i].Resource, "project:")
			var p models.Project
			// 尝试按 UUID 或 ID 标识符查询
			if err := r.db.Select("project_name").Where("id = ? OR project_id_string = ?", pID, pID).First(&p).Error; err == nil {
				logs[i].ResourceName = "项目: " + p.ProjectName
			}
		} else if strings.HasPrefix(logs[i].Resource, "user:") {
			uID := strings.TrimPrefix(logs[i].Resource, "user:")
			var u models.User
			if err := r.db.Select("username").Where("id = ?", uID).First(&u).Error; err == nil {
				logs[i].ResourceName = "用户: " + u.Username
			}
		}
	}

	c.JSON(200, gin.H{
		"message": "获取成功",
		"code":    constant.SuccessCode,
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}
