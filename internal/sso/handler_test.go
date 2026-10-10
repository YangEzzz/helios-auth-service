package sso

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/session"
)

func TestRedirectValidation(t *testing.T) {
	for _, raw := range []string{"https://nexus.example.com/api/callback", "http://localhost:5173/callback", "http://127.0.0.1:5173/callback"} {
		if !validRedirect(raw) {
			t.Errorf("rejected %s", raw)
		}
	}
	for _, raw := range []string{"//evil.example/cb", "http://evil.example/cb", "https://user:pass@nexus.example/cb", "https://nexus.example/cb#fragment", "javascript:alert(1)"} {
		if validRedirect(raw) {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestCodeExchangeBindingAndConcurrentConsumption(t *testing.T) {
	dsn := os.Getenv("HELIOS_SSO_TEST_DSN")
	if dsn == "" {
		t.Skip("set HELIOS_SSO_TEST_DSN to run database integration checks")
	}
	if (!strings.Contains(dsn, "host=127.0.0.1") || !strings.Contains(dsn, "_sso_test")) && os.Getenv("SSO_TEST_ALLOW_EXISTING_DATABASE") != "true" {
		t.Fatal("existing databases require explicit SSO_TEST_ALLOW_EXISTING_DATABASE=true")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	u := models.NewUser("Exchange test", "exchange-"+time.Now().Format("150405.000000000")+"@sso.test", "unused", "", "", "")
	u.Status = constant.UserStatusActive
	p := models.NewProject("Exchange test", "test-"+u.ID.String(), "", "", nil)
	clientID := "test-" + u.ID.String()
	t.Cleanup(func() {
		// Delete only this test's UUID-scoped data, including its audit entry.
		for _, result := range []*gorm.DB{
			db.Delete(&Code{}, "client_id = ?", clientID),
			db.Delete(&Client{}, "id = ?", clientID),
			db.Delete(&models.AuditLog{}, "user_id = ?", u.ID),
			db.Delete(&models.ProjectMembership{}, "user_id = ? AND project_id = ?", u.ID, p.ID),
			db.Delete(&models.Project{}, "id = ?", p.ID),
			db.Delete(&models.User{}, "id = ?", u.ID),
		} {
			if result.Error != nil {
				t.Errorf("test cleanup: %v", result.Error)
			}
		}
	})
	if err = db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	member := models.NewProjectMembership(u.ID, p.ID, "translator")
	if err = db.Create(member).Error; err != nil {
		t.Fatal(err)
	}
	secret, _ := session.Random()
	cl := Client{ID: "test-" + u.ID.String(), ProjectID: p.ID, Name: "Test", SecretHash: session.Hash(secret), Enabled: true, RedirectURIs: pq.StringArray{"https://nexus.example/callback"}}
	if err = db.Create(&cl).Error; err != nil {
		t.Fatal(err)
	}
	verifier, _ := session.Random()
	raw, _ := session.Random()
	code := Code{ID: session.Hash(raw), UserID: u.ID, ClientID: cl.ID, RedirectURI: cl.RedirectURIs[0], Challenge: session.Challenge(verifier), ExpiresAt: time.Now().Add(time.Minute)}
	if err = db.Create(&code).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &Handler{DB: db, Audit: audit.NewService(audit.NewDao(db))}
	r.POST("/exchange", h.Exchange)
	exchange := func(clientID, clientSecret, redirect, proof string) int {
		b, _ := json.Marshal(map[string]string{"code": raw, "redirect_uri": redirect, "code_verifier": proof})
		req := httptest.NewRequest(http.MethodPost, "/exchange", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(clientID, clientSecret)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if status := exchange(cl.ID, "wrong", cl.RedirectURIs[0], verifier); status != 401 {
		t.Fatalf("secret check: %d", status)
	}
	if status := exchange(cl.ID, secret, "https://evil.example/callback", verifier); status != 400 {
		t.Fatalf("redirect binding: %d", status)
	}
	wrong, _ := session.Random()
	if status := exchange(cl.ID, secret, cl.RedirectURIs[0], wrong); status != 400 {
		t.Fatalf("PKCE binding: %d", status)
	}
	// The actual concurrent requests prove row-lock based one-time consumption.
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- exchange(cl.ID, secret, cl.RedirectURIs[0], verifier) }()
	}
	wg.Wait()
	close(results)
	ok, bad := 0, 0
	for status := range results {
		if status == 200 {
			ok++
		} else if status == 400 {
			bad++
		} else {
			t.Fatalf("unexpected exchange status %d", status)
		}
	}
	if ok != 1 || bad != 1 {
		t.Fatalf("expected one success and one rejection: %d/%d", ok, bad)
	}
	raw, _ = session.Random()
	code.ID = session.Hash(raw)
	code.UsedAt = nil
	code.ExpiresAt = time.Now().Add(-time.Second)
	if err = db.Create(&code).Error; err != nil {
		t.Fatal(err)
	}
	if status := exchange(cl.ID, secret, cl.RedirectURIs[0], verifier); status != 400 {
		t.Fatalf("expiry: %d", status)
	}
}
