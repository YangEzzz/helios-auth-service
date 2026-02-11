package user

import (
	"context"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
)

type Service interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	ApproveUser(ctx context.Context, id string) error
	RejectUser(ctx context.Context, id string) error
	GetTotalUserCount(ctx context.Context) (int64, error)
	GetUserCountByStatus(ctx context.Context, status constant.UserStatus) (int64, error)
	GetAllUsers(ctx context.Context, page, pageSize int) ([]*models.User, int64, error)
}

type service struct {
	dao       Dao
	jwtSecret string
}

func NewService(dao Dao, jwtSecret string) Service {
	return &service{dao: dao, jwtSecret: jwtSecret}
}

func (s *service) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.dao.GetUserByID(id)
}

func (s *service) ApproveUser(ctx context.Context, id string) error {
	return s.dao.ApproveUser(id)
}

func (s *service) RejectUser(ctx context.Context, id string) error {
	return s.dao.RejectUser(id)
}

// GetTotalUserCount 获取总用户数
func (s *service) GetTotalUserCount(ctx context.Context) (int64, error) {
	return s.dao.GetTotalUserCount()
}

// GetUserCountByStatus 根据状态获取用户数
func (s *service) GetUserCountByStatus(ctx context.Context, status constant.UserStatus) (int64, error) {
	return s.dao.GetUserCountByStatus(status)
}

// GetAllUsers 获取所有用户列表（支持分页）
// page: 页码（从1开始）
// pageSize: 每页数量
// 返回: 用户列表, 总数, 错误
func (s *service) GetAllUsers(ctx context.Context, page, pageSize int) ([]*models.User, int64, error) {
	// 计算偏移量
	offset := (page - 1) * pageSize

	// 获取用户列表
	users, err := s.dao.GetAllUsers(offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	total, err := s.dao.GetTotalUserCount()
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
