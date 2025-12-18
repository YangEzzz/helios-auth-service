package user

import (
	"context"
	"helios-auth-service/internal/models"
)

type Service interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	ApproveUser(ctx context.Context, id string) error
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
