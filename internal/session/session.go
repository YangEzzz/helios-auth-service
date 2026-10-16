package session

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
)

const CookieName = "helios_session"
const CSRFCookie = "helios_csrf"

const Lifetime = 30 * 24 * time.Hour
const CookieMaxAge = int(Lifetime / time.Second)

type Session struct {
	ID        string    `gorm:"primaryKey;size:64"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	ExpiresAt time.Time `gorm:"not null;index"`
	CreatedAt time.Time
}

func (Session) TableName() string { return "sso_sessions" }

func Random() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func Hash(v string) string { b := sha256.Sum256([]byte(v)); return hex.EncodeToString(b[:]) }
func Challenge(v string) string {
	b := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func Equal(a, b string) bool { return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func Cookie(c *gin.Context, name, value string, age int, secure, httpOnly bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, age, "/", "", secure, httpOnly)
}
func CSRF(c *gin.Context, secure bool) (string, error) {
	value, _ := c.Cookie(CSRFCookie)
	if len(value) != 43 {
		var err error
		value, err = Random()
		if err != nil {
			return "", err
		}
		Cookie(c, CSRFCookie, value, CookieMaxAge, secure, false)
	}
	return value, nil
}
func Guard(c *gin.Context, origin string) bool {
	value, _ := c.Cookie(CSRFCookie)
	if c.GetHeader("Origin") != origin || !Equal(value, c.GetHeader("X-CSRF-Token")) {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "请求校验失败，请刷新页面重试"})
		return false
	}
	return true
}
func Origin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
func Create(c *gin.Context, db *gorm.DB, userID uuid.UUID, secure bool) error {
	value, err := Random()
	if err != nil {
		return err
	}
	old, _ := c.Cookie(CookieName)
	err = db.Transaction(func(tx *gorm.DB) error {
		if old != "" {
			if err := tx.Delete(&Session{}, "id = ?", Hash(old)).Error; err != nil {
				return err
			}
		}
		return tx.Create(&Session{ID: Hash(value), UserID: userID, ExpiresAt: time.Now().Add(Lifetime)}).Error
	})
	if err == nil {
		Cookie(c, CookieName, value, CookieMaxAge, secure, true)
	}
	return err
}
func User(c *gin.Context, db *gorm.DB) (*models.User, error) {
	value, err := c.Cookie(CookieName)
	if err != nil {
		return nil, err
	}
	var s Session
	if err = db.Where("id = ? AND expires_at > ?", Hash(value), time.Now()).First(&s).Error; err != nil {
		return nil, err
	}
	var u models.User
	if err = db.First(&u, "id = ?", s.UserID).Error; err != nil {
		return nil, err
	}
	if u.Status != constant.UserStatusActive {
		return nil, errors.New("inactive user")
	}
	return &u, nil
}
func Revoke(c *gin.Context, db *gorm.DB, secure bool) error {
	value, _ := c.Cookie(CookieName)
	if value != "" {
		if err := db.Delete(&Session{}, "id = ?", Hash(value)).Error; err != nil {
			return err
		}
	}
	Cookie(c, CookieName, "", -1, secure, true)
	return nil
}
