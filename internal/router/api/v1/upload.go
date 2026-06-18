package v1

import (
	"fmt"
	"helios-auth-service/internal/constant"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxAvatarSize = 10 * 1024 * 1024

var allowedAvatarTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
}

// UploadAvatar 处理用户头像上传
func UploadAvatar(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(200, gin.H{"message": "未找到上传文件", "code": constant.ErrorCode})
		return
	}
	if file.Size <= 0 || file.Size > maxAvatarSize {
		c.JSON(200, gin.H{"message": "头像大小不能超过 10MB", "code": constant.ErrorCode})
		return
	}

	// 1. 简单的文件类型校验
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		c.JSON(200, gin.H{"message": "不支持的文件格式", "code": constant.ErrorCode})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(200, gin.H{"message": "文件读取失败: " + err.Error(), "code": constant.ErrorCode})
		return
	}
	defer src.Close()

	header := make([]byte, 512)
	n, err := io.ReadFull(src, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		c.JSON(200, gin.H{"message": "文件读取失败: " + err.Error(), "code": constant.ErrorCode})
		return
	}
	contentType := http.DetectContentType(header[:n])
	if _, ok := allowedAvatarTypes[contentType]; !ok {
		c.JSON(200, gin.H{"message": "不支持的文件内容类型", "code": constant.ErrorCode})
		return
	}

	// 2. 生成新文件名
	newFileName := uuid.New().String() + ext
	savePath := filepath.Join("./uploads/avatars", newFileName)

	// 3. 保存文件
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(200, gin.H{"message": "文件保存失败: " + err.Error(), "code": constant.ErrorCode})
		return
	}

	// 4. 返回 URL (暂时使用相对路径，前端补齐 baseURL 或者后端补齐)
	// 建议返回 /uploads/avatars/xxx.jpg
	avatarURL := fmt.Sprintf("/uploads/avatars/%s", newFileName)

	c.JSON(200, gin.H{
		"message": "上传成功",
		"code":    constant.SuccessCode,
		"data": gin.H{
			"url": avatarURL,
		},
	})
}
