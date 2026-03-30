package constant

type UserRole string

const (
	// UserRoleSuperAdmin 超级管理员
	UserRoleSuperAdmin = "super_admin"
	// UserRoleAdmin 管理员
	UserRoleAdmin = "admin"
	// UserRoleUser 普通用户
	UserRoleUser = "user"
)

type UserStatus string

const (
	// UserStatusActive 激活
	UserStatusActive = "active"
	// UserStatusInactive 未激活
	UserStatusInactive = "inactive"
	// UserStatusLocked 锁定
	UserStatusLocked = "locked"
	// UserStatusPending 待审核
	UserStatusPending = "pending_approval"
	// UserStatusRejected 已拒绝
	UserStatusRejected = "rejected"
)

const (
	SuccessCode          = 200
	ErrorCode            = 500
	NotProjectMemberCode = 40301
)
