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
	// 对密码进行hash处理
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := models.NewUser(name, email, hashedPassword)
	err = s.dao.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (string, error) {
	fmt.Println("Login user:", email)
	user, err := s.dao.GetUserByEmail(email)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// 使用bcrypt验证密码
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", fmt.Errorf("invalid password")
	}

	token, err := utils.GenerateJWT(user.ID, s.jwtSecret, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	fmt.Println("Login successful:", token)
	fmt.Println("Login successful:", user)
	return token, nil
}
