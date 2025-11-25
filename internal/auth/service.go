package auth

import (
	"context"
	"fmt"
	"helios-auth-service/internal/models"
)

type Service interface {
	// Register 用户注册
	Register(ctx context.Context, email, name, password string) (*models.User, error)
	// Login 用户登录（返回 JWT Token）
	Login(ctx context.Context, email, password string) (string, error)
}

type service struct {
	dao       Dao
	jwtSecret string
}

func NewService(dao Dao, jwtSecret string) Service {
	return &service{dao: dao, jwtSecret: jwtSecret}
}

func (s *service) Register(ctx context.Context, email, name, password string) (*models.User, error) {
	fmt.Println("Registering user:", email, name, password)
	return nil, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, error) {
	return "", nil
}
