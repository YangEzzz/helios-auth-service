package utils

import (
	"errors"
	"fmt"
	"helios-auth-service/internal/constant"
	"strings"

	"github.com/go-playground/validator/v10"
)

// 中文字段名映射
var fieldNameMap = map[string]string{
	"Name":     "姓名",
	"Email":    "邮箱",
	"Password": "密码",
}

// GetValidationError 将 validator 错误转换为友好的中文提示
func GetValidationError(err error) string {
	fmt.Println("err:", err)
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, e := range validationErrors {
			fieldName := getFieldName(e.Field())
			return getErrorMessage(fieldName, e.Tag(), e.Param())
		}
	}
	// 2. 处理JSON解析错误
	if strings.Contains(err.Error(), "invalid character") ||
		strings.Contains(err.Error(), "unexpected end of JSON") {
		return "JSON格式错误"
	}

	// 3. 处理数据库错误
	if strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "UNIQUE constraint") {
		return "该邮箱已被注册"
	}

	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "database") {
		return "数据库连接失败，请稍后重试"
	}

	// 4. 处理业务逻辑错误 (Sentinel Errors)
	if errors.Is(err, constant.ErrUserNotFound) || errors.Is(err, constant.ErrRecordNotFound) || errors.Is(err, constant.ErrProjectNotFound) {
		return "记录不存在"
	}
	if errors.Is(err, constant.ErrInvalidPassword) {
		return "密码错误"
	}
	if errors.Is(err, constant.ErrEmailAlreadyExists) {
		return "该邮箱已被注册"
	}

	return "请求参数格式错误"
}

// getFieldName 获取字段的中文名称
func getFieldName(field string) string {
	if name, ok := fieldNameMap[field]; ok {
		return name
	}
	return field
}

// getErrorMessage 根据验证标签返回对应的错误信息
func getErrorMessage(fieldName, tag, param string) string {
	fmt.Println("tag:", tag, "param:", param)
	switch tag {
	case "required":
		return fmt.Sprintf("%s不能为空", fieldName)
	case "email":
		return fmt.Sprintf("%s格式不正确", fieldName)
	case "min":
		return fmt.Sprintf("%s长度不能少于%s位", fieldName, param)
	case "max":
		return fmt.Sprintf("%s长度不能超过%s位", fieldName, param)
	case "len":
		return fmt.Sprintf("%s长度必须为%s位", fieldName, param)
	default:
		return fmt.Sprintf("%s验证失败", fieldName)
	}
}
