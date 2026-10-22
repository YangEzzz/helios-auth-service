package sso

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestProjectImageAuth(t *testing.T) {
	userID := uuid.New()
	for _, tc := range []struct {
		name, user string
		err        error
		want       int
	}{
		{"active member", userID.String(), nil, 200},
		{"missing user", "", nil, 400},
		{"invalid user", "not-a-uuid", nil, 400},
		{"invalid project credentials", userID.String(), errImageCredentials, 401},
		{"inactive or removed member", userID.String(), gorm.ErrRecordNotFound, 403},
		{"database unavailable", userID.String(), errors.New("database unavailable"), 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/image", imageAuth(func(c *gin.Context) { c.AbortWithStatus(401) }, func(c *gin.Context, id uuid.UUID) error {
				if id != userID {
					t.Fatal("user identity changed")
				}
				client, secret, ok := c.Request.BasicAuth()
				if !ok || client != "atlas" || secret != "fixture-secret" {
					t.Fatal("project credentials missing")
				}
				return tc.err
			}), func(c *gin.Context) {
				if c.GetString("user_id") != userID.String() {
					t.Fatal("missing authenticated user")
				}
				c.Status(200)
			})
			req := httptest.NewRequest("POST", "/image", nil)
			req.SetBasicAuth("atlas", "fixture-secret")
			req.Header.Set("X-Helios-User-ID", tc.user)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
		})
	}
	r := gin.New()
	r.POST("/image", imageAuth(func(c *gin.Context) { c.AbortWithStatus(401) }, func(*gin.Context, uuid.UUID) error { t.Fatal("anonymous request verified as project"); return nil }), func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/image", nil))
	if w.Code != 401 {
		t.Fatalf("anonymous request accepted: %d", w.Code)
	}
}
