package user

import (
	"context"
	"errors"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
)

type Service interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	ApproveUser(ctx context.Context, id string) error
	RejectUser(ctx context.Context, id string) error
	LockUser(ctx context.Context, id string) error
	UnlockUser(ctx context.Context, id string) error
	GetTotalUserCount(ctx context.Context) (int64, error)
	GetUserCountByStatus(ctx context.Context, status constant.UserStatus) (int64, error)
	// GetAllUsers 获取所有用户列表（支持分页）
	GetAllUsers(ctx context.Context, page, pageSize int) ([]*models.User, int64, error)
	// SetUserRole 设置用户角色 (仅管理员)
	SetUserRole(ctx context.Context, operatorID, targetUserID string, newRole constant.UserRole) error
	UpdateAvatar(ctx context.Context, userID, avatarURL string) error
}

type service struct {
	dao          Dao
	jwtSecret    string
	auditService audit.Service
}

func NewService(dao Dao, jwtSecret string, auditService audit.Service) Service {
	return &service{dao: dao, jwtSecret: jwtSecret, auditService: auditService}
}

func (s *service) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user, err := s.dao.GetUserByID(id)
	if err != nil {
		if errors.Is(err, constant.ErrUserNotFound) {
			return nil, constant.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *service) ApproveUser(ctx context.Context, id string) error {
	if err := s.dao.ApproveUser(id); err != nil {
		return err
	}
	_ = s.auditService.LogAction(ctx, nil, "approve_user", "user:"+id, "Approved user", "")
	return nil
}

func (s *service) RejectUser(ctx context.Context, id string) error {
	if err := s.dao.RejectUser(id); err != nil {
		return err
	}
	_ = s.auditService.LogAction(ctx, nil, constant.ActionRejectUser, "user:"+id, "Rejected user", "")
	return nil
}

func (s *service) LockUser(ctx context.Context, id string) error {
	if err := s.dao.LockUser(id); err != nil {
		return err
	}
	_ = s.auditService.LogAction(ctx, nil, constant.ActionLockUser, "user:"+id, "Locked user", "")
	return nil
}

func (s *service) UnlockUser(ctx context.Context, id string) error {
	if err := s.dao.UnlockUser(id); err != nil {
		return err
	}
	_ = s.auditService.LogAction(ctx, nil, constant.ActionUnlockUser, "user:"+id, "Unlocked user", "")
	return nil
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

// SetUserRole 设置用户角色
func (s *service) SetUserRole(ctx context.Context, operatorID, targetUserID string, newRole constant.UserRole) error {
	// 1. 获取操作者信息
	operator, err := s.dao.GetUserByID(operatorID)
	if err != nil {
		if errors.Is(err, constant.ErrUserNotFound) {
			return errors.New("operator not found")
		}
		return err
	}

	// 2. 检查权限 (必须是 Admin 或 SuperAdmin)
	if operator.Role != constant.UserRoleAdmin && operator.Role != constant.UserRoleSuperAdmin {
		return constant.ErrUnauthorized
	}

	// 3. 更新角色
	if err := s.dao.UpdateUserRole(targetUserID, newRole); err != nil {
		return err
	}

	// 4. 记录审计日志
	_ = s.auditService.LogAction(ctx, nil, "set_user_role", "user:"+targetUserID, "Set role to "+string(newRole)+" by "+operator.Username, "")

	return nil
}

func (s *service) UpdateAvatar(ctx context.Context, userID, avatarURL string) error {
	if err := s.dao.UpdateAvatar(userID, avatarURL); err != nil {
		return err
	}
	// 记录审计日志
	_ = s.auditService.LogAction(ctx, nil, "update_avatar", "user:"+userID, "Updated profile avatar", "")
	return nil
}
