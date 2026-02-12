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
)
