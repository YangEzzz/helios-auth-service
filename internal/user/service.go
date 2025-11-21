package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"helios-auth-service/internal/models"
	"helios-auth-service/internal/repository"
	"helios-auth-service/internal/utils"
)

// UserService 定义用户服务接口
type UserService interface {
	// RegisterUser 用户注册
	RegisterUser(ctx context.Context, email, password string) (*models.User, error)
	// Login 用户登录（返回 JWT Token）
	Login(ctx context.Context, email, password string) (string, error)
	// GetUserByID 通过 ID 获取用户信息
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
}

type userService struct {
	userRepo  repository.UserRepository
	jwtSecret string
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, jwtSecret string) UserService {
	return &userService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// RegisterUser 用户注册
func (s *userService) RegisterUser(ctx context.Context, email, password string) (*models.User, error) {
	// 1. 检查邮箱是否已注册
	_, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil { // 用户已存在
		return nil, errors.New("email already registered")
	}

	// 2. 哈希密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 3. 创建用户模型（暂时使用 email 作为 username）
	user := models.NewUser(email, email, hashedPassword)

	// 4. 创建用户
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login 用户登录
func (s *userService) Login(ctx context.Context, email, password string) (string, error) {
	// 1. 获取用户信息
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid credentials") // 避免暴露用户是否存在
	}

	// 2. 验证密码
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", errors.New("invalid credentials")
	}

	// 3. 生成 JWT Token
	token, err := utils.GenerateJWT(user.ID, s.jwtSecret, time.Hour*24) // Token有效期24小时
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return token, nil
}

// GetUserByID 通过 ID 获取用户信息
func (s *userService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	// 解析 UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
