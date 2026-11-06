package sso

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/organization"
	"helios-auth-service/internal/session"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSSOOrganizationIdentityPostgres(t *testing.T) {
	if os.Getenv("ORG_TEST_DB") != "1" {
		t.Skip("set ORG_TEST_DB=1 for isolated transactional PostgreSQL checks")
	}
	_ = godotenv.Load("../../.env")
	cfg := config.LoadConfig()
	db, err := database.NewConnection(cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseName, cfg.DatabaseSSLMode)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	schema := "sso_org_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	for _, sql := range []string{
		"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema + ", public",
		"CREATE TABLE users(id uuid PRIMARY KEY,username text,email text,status text,role text)",
		"CREATE TABLE projects(id uuid PRIMARY KEY,project_id_string text)",
		"CREATE TABLE project_memberships(id uuid PRIMARY KEY,user_id uuid,project_id uuid,role_in_project text)",
		"CREATE TABLE audit_logs(id uuid PRIMARY KEY,user_id uuid,action text,resource text,details text,ip_address text,created_at timestamptz)",
	} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := organization.Migrate(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&Client{}, &Code{}); err != nil {
		t.Fatal(err)
	}
	uid, pid, other := uuid.New(), uuid.New(), uuid.New()
	for _, sql := range []struct {
		q    string
		args []any
	}{
		{"INSERT INTO users VALUES (?, 'SSO fixture','fixture@example.test','active','user')", []any{uid}},
		{"INSERT INTO projects VALUES (?,'fixture-project')", []any{pid}},
		{"INSERT INTO project_memberships VALUES (?,?,?,'reader')", []any{uuid.New(), uid, pid}},
		{"INSERT INTO users VALUES (?, 'Other fixture','other@example.test','active','user')", []any{other}},
	} {
		if err := tx.Exec(sql.q, sql.args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	root := organization.Department{ID: uuid.New(), Code: uuid.NewString(), Name: "技术部", Status: "active", Version: 1}
	rd := organization.Department{ID: uuid.New(), ParentID: &root.ID, Code: uuid.NewString(), Name: "研发部", Status: "active", Version: 1}
	qa := organization.Department{ID: uuid.New(), ParentID: &root.ID, Code: uuid.NewString(), Name: "测试部", Status: "active", Version: 1}
	for _, d := range []*organization.Department{&root, &rd, &qa} {
		if err := tx.Create(d).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Create(&organization.Membership{DepartmentID: rd.ID, UserID: uid, IsManager: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&organization.Membership{DepartmentID: qa.ID, UserID: other}).Error; err != nil {
		t.Fatal(err)
	}
	secret := "fixture-secret"
	client := Client{ID: "fixture-client", ProjectID: pid, Name: "Fixture", SecretHash: session.Hash(secret), Enabled: true, RedirectURIs: pq.StringArray{"https://fixture.example/callback"}}
	if err := tx.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	h := Handler{DB: tx, Audit: audit.NewService(audit.NewDao(tx))}
	r := gin.New()
	r.POST("/identity", h.Identity)
	r.POST("/exchange", h.Exchange)
	type response struct {
		Data struct {
			User struct {
				ID          uuid.UUID               `json:"id"`
				Departments []models.UserDepartment `json:"departments"`
			} `json:"user"`
			Organization *OrganizationInfo `json:"organization"`
			Project      string            `json:"project_id_string"`
			Role         string            `json:"role_in_project"`
		} `json:"data"`
	}
	call := func(path string, body any, password string, want int) response {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(client.ID, password)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s got %d want %d: %s", path, w.Code, want, w.Body)
		}
		if strings.Contains(w.Body.String(), "secret_hash") || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "member_count") {
			t.Fatal("administrative data exposed")
		}
		var result response
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	body := gin.H{"user_id": uid.String()}
	call("/identity", body, "wrong", 401)
	call("/identity", gin.H{"user_id": other.String()}, secret, 403)
	initial := call("/identity", body, secret, 200)
	if initial.Data.Organization == nil || len(initial.Data.Organization.Departments) != 3 || len(initial.Data.User.Departments) != 1 || initial.Data.User.Departments[0].ID != rd.ID || !initial.Data.User.Departments[0].IsManager || initial.Data.Role != "reader" {
		t.Fatalf("unexpected identity: %+v", initial)
	}
	verifier, _ := session.Random()
	rawCode, _ := session.Random()
	code := Code{ID: session.Hash(rawCode), UserID: uid, ClientID: client.ID, RedirectURI: client.RedirectURIs[0], Challenge: session.Challenge(verifier), ExpiresAt: time.Now().Add(time.Minute)}
	if err := tx.Create(&code).Error; err != nil {
		t.Fatal(err)
	}
	exchangeBody := gin.H{"code": rawCode, "redirect_uri": client.RedirectURIs[0], "code_verifier": verifier}
	exchanged := call("/exchange", exchangeBody, secret, 200)
	if len(exchanged.Data.User.Departments) != 1 || exchanged.Data.Organization == nil {
		t.Fatal("exchange omitted organization")
	}
	call("/exchange", exchangeBody, secret, 400)
	if err := tx.Create(&organization.Membership{DepartmentID: root.ID, UserID: uid}).Error; err != nil {
		t.Fatal(err)
	}
	multiple := call("/identity", body, secret, 200)
	if len(multiple.Data.User.Departments) != 2 {
		t.Fatal("multiple equal department memberships missing")
	}
	if err := tx.Exec("DELETE FROM department_memberships WHERE user_id=? AND department_id=?", uid, root.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("UPDATE department_memberships SET department_id=?,is_manager=false WHERE user_id=?", qa.ID, uid).Error; err != nil {
		t.Fatal(err)
	}
	moved := call("/identity", body, secret, 200)
	if moved.Data.User.Departments[0].ID != qa.ID || moved.Data.User.Departments[0].IsManager || moved.Data.Organization.Version != initial.Data.Organization.Version {
		t.Fatal("live membership refresh or directory-only version incorrect")
	}
	if err := tx.Model(&qa).Update("name", "测试二部").Error; err != nil {
		t.Fatal(err)
	}
	renamed := call("/identity", body, secret, 200)
	if renamed.Data.Organization.Version == initial.Data.Organization.Version || renamed.Data.User.Departments[0].Name != "测试二部" {
		t.Fatal("department rename not refreshed")
	}
	if err := tx.Exec("DELETE FROM department_memberships WHERE user_id=?", uid).Error; err != nil {
		t.Fatal(err)
	}
	empty := call("/identity", body, secret, 200)
	if empty.Data.User.Departments == nil || len(empty.Data.User.Departments) != 0 {
		t.Fatal("empty legacy memberships must be []")
	}
	if err := tx.Exec("UPDATE users SET status='locked' WHERE id=?", uid).Error; err != nil {
		t.Fatal(err)
	}
	call("/identity", body, secret, 403)
	if err := tx.Exec("UPDATE users SET status='active' WHERE id=?", uid).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("DELETE FROM project_memberships WHERE user_id=?", uid).Error; err != nil {
		t.Fatal(err)
	}
	call("/identity", body, secret, 403)
}
