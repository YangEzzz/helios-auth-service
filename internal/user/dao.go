package user

import "helios-auth-service/internal/models"

type Dao interface {
	// GetUserByEmail 通过邮箱获取用户
	GetUserByEmail(email string) (*models.User, error)
	// GetUserByID 通过ID获取用户
	GetUserByID(id string) (*models.User, error)
}
