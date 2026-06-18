package auth

import (
	"context"
	"errors"
	"fmt"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/utils"
	"time"
)

type Service interface {
	// Register 用户注册
	Register(ctx context.Context, email, name, password, department, reason, avatar string) (*models.User, error)
	// Login 用户登录（返回 用户信息和 JWT Token）
	Login(ctx context.Context, email, password string) (*models.User, string, error)
}

type service struct {
	dao       Dao
	jwtSecret string
}

func NewService(dao Dao, jwtSecret string) Service {
	return &service{dao: dao, jwtSecret: jwtSecret}
}

func (s *service) Register(ctx context.Context, email, name, password, department, reason, avatar string) (*models.User, error) {
	// 对密码进行hash处理
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := models.NewUser(name, email, hashedPassword, department, reason, avatar)
	err = s.dao.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	fmt.Println("Login user:", email)
	user, err := s.dao.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, constant.ErrUserNotFound) {
			return nil, "", constant.ErrUserNotFound
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	// 使用bcrypt验证密码
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", constant.ErrInvalidPassword
	}

	// 状态拦截：只有状态为 Active 的用户允许登录
	switch user.Status {
	case constant.UserStatusPending:
		return nil, "", constant.ErrUserPendingApproval
	case constant.UserStatusLocked:
		return nil, "", constant.ErrUserLocked
	case constant.UserStatusRejected:
		return nil, "", constant.ErrUserRejected
	case constant.UserStatusInactive:
		return nil, "", constant.ErrUserPendingApproval // 或者自定义未激活逻辑
	case constant.UserStatusActive:
		// 允许通过
	default:
		return nil, "", constant.ErrUnauthorized
	}
	// 十天
	token, err := utils.GenerateJWT(user.ID, s.jwtSecret, 10*24*time.Hour)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	fmt.Println("Login successful:", token)
	fmt.Println("Login successful:", user)
	return user, token, nil
}
