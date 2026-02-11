package user

import (
	"helios-auth-service/internal/constant"
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
	// RejectUser 拒绝用户
	RejectUser(id string) error
	// GetTotalUserCount 获取总用户数
	GetTotalUserCount() (int64, error)
	// GetUserCountByStatus 根据状态获取用户数
	GetUserCountByStatus(status constant.UserStatus) (int64, error)
	// GetAllUsers 获取所有用户列表（支持分页）
	GetAllUsers(offset, limit int) ([]*models.User, error)
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
	return u.db.Model(&models.User{}).Where("id = ?", id).Update("status", constant.UserStatusActive).Error
}

func (u *userDao) RejectUser(id string) error {
	return u.db.Model(&models.User{}).Where("id = ?", id).Update("status", constant.UserStatusRejected).Error
}

// GetTotalUserCount 获取总用户数
func (u *userDao) GetTotalUserCount() (int64, error) {
	var count int64
	err := u.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

// GetUserCountByStatus 根据状态获取用户数
func (u *userDao) GetUserCountByStatus(status constant.UserStatus) (int64, error) {
	var count int64
	err := u.db.Model(&models.User{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// GetAllUsers 获取所有用户列表（支持分页）
func (u *userDao) GetAllUsers(offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := u.db.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
