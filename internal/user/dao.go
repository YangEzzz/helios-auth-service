package user

import (
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type userDao struct {
	db *gorm.DB
}

type Dao interface {
	// GetUserByEmail 通过邮箱获取用户
	GetUserByEmail(email string) (*models.User, error)
	// GetUserByID 通过ID获取用户
	GetUserByID(id string) (*models.User, error)
	// ApproveUser 审核用户
	ApproveUser(id string) error
}

func NewDao(db *gorm.DB) Dao {
	return &userDao{db: db}
}

func (u *userDao) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := u.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *userDao) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := u.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *userDao) ApproveUser(id string) error {
	return u.db.Model(&models.User{}).Where("id = ?", id).Update("status", models.UserStatusActive).Error
}
