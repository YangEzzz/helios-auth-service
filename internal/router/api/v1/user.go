package v1

import (
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/user"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserRouter struct {
	userService user.Service
}

func NewUserRouter(userService user.Service) *UserRouter {
	return &UserRouter{
		userService: userService,
	}
}

func InitUserRouter(r *gin.RouterGroup, db *gorm.DB, jwtSecret string) {
	userDao := user.NewDao(db)
	userService := user.NewService(userDao, jwtSecret)
	userRouter := NewUserRouter(userService)

	projectGroup := r.Group("")
	projectGroup.Use(middleware.AuthMiddleware(jwtSecret))
	{
		projectGroup.GET("/users", userRouter.GetUserByID)
	}
}

func (r *UserRouter) GetUserByID(c *gin.Context) {
	id := c.Query("id")
	users, err := r.userService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error(), "code": constant.ErrorCode})
		return
	}
	c.JSON(200, gin.H{"user": users, "code": constant.SuccessCode})
}
