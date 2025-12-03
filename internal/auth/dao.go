package auth

import (
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type authDao struct {
	db *gorm.DB
}

type Dao interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
}

func NewDao(db *gorm.DB) Dao {
	return &authDao{db: db}
}

func (a *authDao) CreateUser(user *models.User) error {
	return a.db.Create(user).Error
}

func (a *authDao) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := a.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
