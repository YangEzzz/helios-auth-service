package v1

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"helios-auth-service/internal/constant"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadImage stores shared images. Uploads require authentication; returned URLs
// are public and must not be used for private documents or credentials.
func UploadImage(c *gin.Context) {
	const maxImageSize = 10 * 1024 * 1024
	// Bound the entire multipart request as well as the individual file.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageSize+64*1024)
	err := c.Request.ParseMultipartForm(maxImageSize + 64*1024)
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的上传请求，请使用 multipart/form-data，图片不能超过 10MB"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未找到上传文件"})
		return
	}
	if file.Size <= 0 || file.Size > maxImageSize {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "图片不能为空且不能超过 10MB"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片读取失败"})
		return
	}
	defer src.Close()
	header := make([]byte, 512)
	n, err := io.ReadFull(src, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片读取失败"})
		return
	}
	contentType := http.DetectContentType(header[:n])
	extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif", "image/webp": ".webp"}
	ext, ok := extensions[contentType]
	originalExt := strings.ToLower(filepath.Ext(file.Filename))
	if !ok || (originalExt != ext && !(contentType == "image/jpeg" && originalExt == ".jpeg")) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持 JPG、PNG、GIF、WebP，文件扩展名需与内容类型一致"})
		return
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "图片读取失败"})
		return
	}
	content, err := io.ReadAll(io.LimitReader(src, maxImageSize+1))
	if err != nil || len(content) != int(file.Size) {
		c.JSON(400, gin.H{"code": 400, "message": "图片读取失败或大小不匹配"})
		return
	}
	digest := fmt.Sprintf("%x", md5.Sum(content))
	if supplied := c.PostForm("md5"); supplied != "" && !strings.EqualFold(supplied, digest) {
		c.JSON(400, gin.H{"code": 400, "message": "图片 MD5 校验失败"})
		return
	}
	if err := os.MkdirAll("./uploads/images", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片保存失败"})
		return
	}
	name := digest + ext
	savePath := filepath.Join("./uploads/images", name)
	reused, err := saveSharedImage(savePath, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": constant.SuccessCode, "message": "上传成功", "data": gin.H{
		"url":      "/uploads/images/" + name,
		"size":     file.Size,
		"mimeType": contentType,
		"md5":      digest,
		"reused":   reused,
	}})
}

// Publish a complete temporary file atomically without overwriting an existing
// image. Concurrent requests and processes sharing this directory use one file.
func saveSharedImage(path string, content []byte) (bool, error) {
	matchExisting := func() (bool, error) {
		existing, err := os.ReadFile(path)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(existing, content) {
			return false, errors.New("image digest collision or corrupt stored image")
		}
		return true, nil
	}
	if reused, err := matchExisting(); !os.IsNotExist(err) {
		return reused, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".image-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return false, err
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if os.IsExist(err) {
			return matchExisting()
		}
		return false, err
	}
	return false, nil
}

// CheckImage lets authenticated clients skip transferring a known image body.
func CheckImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	var request struct {
		MD5 string `json:"md5"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请提供有效的图片 MD5"})
		return
	}
	digest := strings.ToLower(request.MD5)
	if value, err := hex.DecodeString(digest); err != nil || len(value) != md5.Size {
		c.JSON(400, gin.H{"code": 400, "message": "MD5 必须为 32 位十六进制字符串"})
		return
	}
	for _, imageType := range []struct{ ext, mime string }{
		{".jpg", "image/jpeg"}, {".png", "image/png"}, {".gif", "image/gif"}, {".webp", "image/webp"},
	} {
		name := digest + imageType.ext
		info, err := os.Stat(filepath.Join("./uploads/images", name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			c.JSON(500, gin.H{"code": 500, "message": "图片查询失败"})
			return
		}
		c.JSON(200, gin.H{"code": 200, "message": "图片已存在", "data": gin.H{
			"exists": true, "url": "/uploads/images/" + name, "md5": digest,
			"size": info.Size(), "mimeType": imageType.mime, "reused": true,
		}})
		return
	}
	c.JSON(200, gin.H{"code": 200, "message": "图片不存在，请上传", "data": gin.H{"exists": false, "md5": digest}})
}

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
