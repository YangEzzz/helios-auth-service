package auth

import (
	"context"
	"fmt"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/utils"
	"time"
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
	user := models.NewUser(name, email, password)
	err := s.dao.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, error) {
	fmt.Println("Login user:", email, password)
	user, err := s.dao.GetUserByEmail(email)
	if err != nil {
		return "Not Found", err
	}
	if user.PasswordHash != password {
		return "Invalid Password", nil
	}
	token, err := utils.GenerateJWT(user.ID, s.jwtSecret, 24*time.Hour)
	if err != nil {
		return "", err
	}
	fmt.Println("Login successful:", token)
	fmt.Println("Login successful:", user)
	return token, nil
}
