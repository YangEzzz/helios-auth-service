package user

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
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
	ApproveUser(id string, departmentIDs []uuid.UUID, operatorID string) error
	// RejectUser 拒绝用户
	RejectUser(id string) error
	// LockUser 锁定用户
	LockUser(id string) error
	// UnlockUser 解锁用户
	UnlockUser(id string) error
	// GetTotalUserCount 获取总用户数
	GetTotalUserCount() (int64, error)
	// GetUserCountByStatus 根据状态获取用户数
	GetUserCountByStatus(status constant.UserStatus) (int64, error)
	// GetAllUsers 获取所有用户列表（支持分页）
	GetAllUsers(offset, limit int) ([]*models.User, error)
	// UpdateUserRole 更新用户角色
	UpdateUserRole(id string, role constant.UserRole) error
	// UpdateAvatar 更新用户头像
	UpdateAvatar(id string, avatar string) error
}

func NewDao(db *gorm.DB) Dao {
	return &userDao{db: db}
}

func (u *userDao) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := u.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (u *userDao) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := u.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constant.ErrUserNotFound
		}
		return nil, err
	}
	if err := u.attachDepartments([]*models.User{&user}); err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *userDao) ApproveUser(id string, departmentIDs []uuid.UUID, operatorID string) error {
	if len(departmentIDs) == 0 {
		return errors.New("审批通过前必须分配至少一个部门")
	}
	actor, err := uuid.Parse(operatorID)
	if err != nil {
		return errors.New("无效的操作人")
	}
	if _, err := uuid.Parse(id); err != nil {
		return errors.New("无效的用户标识")
	}
	unique := make(map[uuid.UUID]bool)
	for _, departmentID := range departmentIDs {
		if departmentID == uuid.Nil {
			return errors.New("请选择有效部门")
		}
		unique[departmentID] = true
	}
	return u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(72410831)").Error; err != nil {
			return err
		}
		var target models.User
		if err := tx.First(&target, "id=?", id).Error; err != nil {
			return err
		}
		if target.Status != constant.UserStatusPending {
			return errors.New("该申请已被处理，请刷新列表")
		}
		for departmentID := range unique {
			var count int64
			if err := tx.Table("departments").Where("id=? AND status='active'", departmentID).Count(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return errors.New("所选部门已停用或不存在，请重新选择")
			}
			if err := tx.Exec("INSERT INTO department_memberships (department_id,user_id,is_manager) VALUES (?,?,false) ON CONFLICT (department_id,user_id) DO NOTHING", departmentID, id).Error; err != nil {
				return err
			}
			if err := tx.Exec("UPDATE departments SET version=version+1,updated_at=now() WHERE id=?", departmentID).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&models.User{}).Where("id=? AND status=?", id, constant.UserStatusPending).Update("status", constant.UserStatusActive)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("该申请已被处理，请刷新列表")
		}
		details, _ := json.Marshal(map[string]any{"department_ids": departmentIDs, "before_status": target.Status, "after_status": constant.UserStatusActive})
		return tx.Create(models.NewAuditLog(&actor, "approve_user", "user:"+id, string(details), "")).Error
	})
}

func (u *userDao) RejectUser(id string) error {
	result := u.db.Model(&models.User{}).Where("id=? AND status=?", id, constant.UserStatusPending).Update("status", constant.UserStatusRejected)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("该申请已被处理，请刷新列表")
	}
	return nil
}

func (u *userDao) LockUser(id string) error {
	result := u.db.Model(&models.User{}).Where("id=? AND status=?", id, constant.UserStatusActive).Update("status", constant.UserStatusLocked)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("只有正常账户可以锁定，请刷新列表")
	}
	return nil
}

func (u *userDao) UnlockUser(id string) error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(72410831)").Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Table("department_memberships").Where("user_id=?", id).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("请先为用户分配部门，再解锁账户")
		}
		result := tx.Model(&models.User{}).Where("id=? AND status=?", id, constant.UserStatusLocked).Update("status", constant.UserStatusActive)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("只有锁定账户可以解锁，请刷新列表")
		}
		return nil
	})
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
	if err := u.attachDepartments(users); err != nil {
		return nil, err
	}
	return users, nil
}

func (u *userDao) attachDepartments(users []*models.User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]string, 0, len(users))
	byID := make(map[string]*models.User, len(users))
	for _, user := range users {
		user.Departments = []models.UserDepartment{}
		ids = append(ids, user.ID.String())
		byID[user.ID.String()] = user
	}
	var rows []struct {
		UserID string
		models.UserDepartment
	}
	if err := u.db.Table("department_memberships m").Select("m.user_id,d.id,d.name,d.status,m.is_manager").Joins("JOIN departments d ON d.id=m.department_id").Where("m.user_id IN ?", ids).Order("d.sort_order,d.name,d.id").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if user := byID[row.UserID]; user != nil {
			user.Departments = append(user.Departments, row.UserDepartment)
		}
	}
	return nil
}

func (u *userDao) UpdateUserRole(id string, role constant.UserRole) error {
	result := u.db.Model(&models.User{}).Where("id = ?", id).Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return constant.ErrUserNotFound
	}
	return nil
}
func (u *userDao) UpdateAvatar(id string, avatar string) error {
	result := u.db.Model(&models.User{}).Where("id = ?", id).Update("avatar", avatar)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return constant.ErrUserNotFound
	}
	return nil
}
