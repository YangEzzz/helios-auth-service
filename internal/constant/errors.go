package constant

import "errors"

var (
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrProjectNotFound 项目不存在
	ErrProjectNotFound = errors.New("project not found")
	// ErrRecordNotFound 记录不存在 (通用)
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidPassword 密码错误
	ErrInvalidPassword = errors.New("invalid password")
	// ErrEmailAlreadyExists 邮箱已存在
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrUnauthorized 未授权
	ErrUnauthorized = errors.New("unauthorized")
	// ErrUserPendingApproval 待审核
	ErrUserPendingApproval = errors.New("账户正在审核中，请联系管理员")
	// ErrUserLocked 账户已锁定
	ErrUserLocked = errors.New("账户已锁定")
	// ErrUserRejected 账户已停用
	ErrUserRejected = errors.New("账户已被停用或拒绝")
)
