package v1

import (
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/dashboard"
	"helios-auth-service/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardRouter struct {
	dashboardService dashboard.Service
}

func NewDashboardRouter(dashboardService dashboard.Service) *DashboardRouter {
	return &DashboardRouter{
		dashboardService: dashboardService,
	}
}

func InitDashboardRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	dashboardDao := dashboard.NewDao(db)
	dashboardService := dashboard.NewService(dashboardDao)
	dashboardRouter := NewDashboardRouter(dashboardService)

	dashboardGroup := r.Group("")
	dashboardGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		dashboardGroup.GET("/dashboard", dashboardRouter.GetDashboard)
	}
}

func (r *DashboardRouter) GetDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"message": "Unauthorized", "code": constant.ErrorCode})
		return
	}

	data, err := r.dashboardService.GetDashboard(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "code": constant.ErrorCode})
		return
	}

	c.JSON(200, gin.H{
		"message": "获取成功",
		"data":    data,
		"code":    constant.SuccessCode,
	})
}
