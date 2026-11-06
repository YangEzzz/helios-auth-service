package organization

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"helios-auth-service/internal/config"
	"helios-auth-service/internal/database"
	"helios-auth-service/internal/user"
	"helios-auth-service/internal/utils"
)

func TestDepartmentInput(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		order      int
		status     string
		valid      bool
	}{
		{"研发部", "rd", 0, "active", true}, {" 研发部 ", "rd-1", 3, "active", true},
		{"无需编码", "", 0, "active", true},
		{" ", "rd", 0, "active", false}, {"研发部", "研发", 0, "active", false},
		{"研发部", "rd", -1, "active", false}, {"研发部", "rd", 0, "unknown", false},
		{strings.Repeat("部", 101), "rd", 0, "active", false},
	} {
		v := departmentInput{Name: tc.name, Code: tc.code, SortOrder: tc.order, Status: tc.status}
		if got := v.validate() == nil; got != tc.valid {
			t.Errorf("%+v: valid=%t", tc, got)
		}
	}
}

// Uses a dedicated transactional schema; no existing users or departments are changed.
func TestOrganizationPostgres(t *testing.T) {
	if os.Getenv("ORG_TEST_DB") != "1" {
		t.Skip("set ORG_TEST_DB=1 to run PostgreSQL integration checks")
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
	schemaName := "org_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	for _, sql := range []string{
		"CREATE SCHEMA " + schemaName,
		"SET LOCAL search_path TO " + schemaName + ", public",
		`CREATE TABLE users (id uuid PRIMARY KEY, username text, email text, status text, role text)`,
		`CREATE TABLE audit_logs (id uuid PRIMARY KEY,user_id uuid,action text,resource text,details text,ip_address text,created_at timestamptz)`,
	} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := Migrate(tx); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(tx); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	admin, member := uuid.New(), uuid.New()
	if err := tx.Exec("INSERT INTO users VALUES (?, 'admin', 'admin@example.test', 'active', 'admin'), (?, 'member', 'member@example.test', 'active', 'user')", admin, member).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r.Group("/api/v1"), tx, "fixture-secret", "http://localhost:5174")
	token, _ := utils.GenerateJWT(admin, "fixture-secret", time.Hour)
	memberToken, _ := utils.GenerateJWT(member, "fixture-secret", time.Hour)
	request := func(method, path string, body any, auth string, want int) json.RawMessage {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "/api/v1/org"+path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body)
		}
		var result struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &result)
		return result.Data
	}
	create := func(name, code string, parent *uuid.UUID) Department {
		t.Helper()
		raw := request("POST", "/departments", departmentInput{Name: name, Code: code, ParentID: parent, Status: "active"}, token, 200)
		var d Department
		if err := json.Unmarshal(raw, &d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	read := func(id uuid.UUID) Department {
		t.Helper()
		var d Department
		if err := tx.First(&d, "id=?", id).Error; err != nil {
			t.Fatal(err)
		}
		return d
	}
	input := func(d Department) departmentInput {
		return departmentInput{Name: d.Name, Code: d.Code, ParentID: d.ParentID, SortOrder: d.SortOrder, Status: d.Status, Version: d.Version}
	}
	request("GET", "/departments", nil, "", 401)
	request("GET", "/departments", nil, memberToken, 403)
	request("POST", "/departments", departmentInput{Name: "forbidden", Code: "forbidden", Status: "active"}, memberToken, 403)
	root := create("技术部", "tech", nil)
	child := create("研发部", "rd", &root.ID)
	sibling := create("测试部", "qa", &root.ID)
	request("POST", "/departments", departmentInput{Name: "研发部", Code: "duplicate", ParentID: &root.ID, Status: "active"}, token, 409)
	request("POST", "/departments", departmentInput{Name: "其他", Code: "rd", Status: "active"}, token, 409)
	generated := create("自动标识部门", "", nil)
	otherGenerated := create("另一个自动标识部门", "", nil)
	if generated.Code == "" || generated.Code == otherGenerated.Code {
		t.Fatal("missing or duplicate generated identifier")
	}
	edit := input(generated)
	edit.Code = ""
	edit.Name = "改名后部门"
	request("PUT", "/departments/"+generated.ID.String(), edit, token, 200)
	if read(generated.ID).Code != generated.Code {
		t.Fatal("editing without a code changed internal identifier")
	}
	v := input(root)
	v.ParentID = &child.ID
	request("PUT", "/departments/"+root.ID.String(), v, token, 400)
	v = input(root)
	v.Status = "disabled"
	request("PUT", "/departments/"+root.ID.String(), v, token, 409)
	url := "/departments/" + child.ID.String() + "/members/" + member.String()
	request("PUT", url, gin.H{"is_manager": true, "version": child.Version}, token, 200)
	child = read(child.ID)
	var membership Membership
	if err := tx.First(&membership, "department_id=? AND user_id=?", child.ID, member).Error; err != nil || !membership.IsManager {
		t.Fatalf("manager not saved: %v", err)
	}
	request("PUT", "/departments/"+sibling.ID.String()+"/members/"+member.String(), gin.H{"is_manager": false, "version": sibling.Version}, token, 200)
	var count int64
	tx.Model(&Membership{}).Where("user_id=?", member).Count(&count)
	if count != 2 {
		t.Fatal("multiple memberships not preserved")
	}
	userInfo, err := user.NewDao(tx).GetUserByID(member.String())
	if err != nil || len(userInfo.Departments) != 2 {
		t.Fatalf("user departments projection: %+v %v", userInfo, err)
	}
	users, err := user.NewDao(tx).GetAllUsers(0, 20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range users {
		if u.ID == member {
			found = len(u.Departments) == 2
		}
	}
	if !found {
		t.Fatal("directory omitted multiple memberships")
	}
	request("PUT", url, gin.H{"is_manager": false, "version": 1}, token, 409)
	v = input(child)
	v.Status = "disabled"
	request("PUT", "/departments/"+child.ID.String(), v, token, 409)
	request("GET", "/departments/"+child.ID.String()+"/members?q=member", nil, token, 200)
	var candidatePage struct {
		Members []Member `json:"members"`
		Total   int64    `json:"total"`
	}
	json.Unmarshal(request("GET", "/departments/"+child.ID.String()+"/members?candidates=true", nil, token, 200), &candidatePage)
	for _, u := range candidatePage.Members {
		if u.UserID == member {
			t.Fatal("existing member returned as candidate")
		}
	}
	var mine []Department
	json.Unmarshal(request("GET", "/me", nil, memberToken, 200), &mine)
	if len(mine) != 2 {
		t.Fatalf("my departments: %d", len(mine))
	}
	request("DELETE", url, gin.H{"version": child.Version}, token, 200)
	tx.Model(&Membership{}).Where("user_id=?", member).Count(&count)
	if count != 1 {
		t.Fatal("removal affected other departments")
	}
	child = read(child.ID)
	v = input(child)
	v.Status = "disabled"
	request("PUT", "/departments/"+child.ID.String(), v, token, 200)
	child = read(child.ID)
	request("PUT", url, gin.H{"version": child.Version}, token, 409)
	v = input(child)
	v.Status = "active"
	request("PUT", "/departments/"+child.ID.String(), v, token, 200)
	// Pagination across >20 eligible users and search.
	for i := 0; i < 23; i++ {
		if err := tx.Exec("INSERT INTO users VALUES (?, 'candidate', ?, 'active','user')", uuid.New(), uuid.NewString()+"@example.test").Error; err != nil {
			t.Fatal(err)
		}
	}
	json.Unmarshal(request("GET", "/departments/"+child.ID.String()+"/members?candidates=true&page=2&q=candidate", nil, token, 200), &candidatePage)
	if candidatePage.Total != 23 || len(candidatePage.Members) != 3 {
		t.Fatalf("pagination: %+v", candidatePage)
	}
	request("GET", "/departments/bad-id/members", nil, token, 400)
	// An approved user cannot lose their final department.
	sibling = read(sibling.ID)
	request("DELETE", "/departments/"+sibling.ID.String()+"/members/"+member.String(), gin.H{"version": sibling.Version}, token, 409)
	if err := tx.Exec("ALTER TABLE users ADD COLUMN updated_at timestamptz").Error; err != nil {
		t.Fatal(err)
	}
	pending := uuid.New()
	if err := tx.Exec("INSERT INTO users (id,username,email,status,role) VALUES (?, 'pending', 'pending@example.test','pending_approval','user')", pending).Error; err != nil {
		t.Fatal(err)
	}
	dao := user.NewDao(tx)
	if err := dao.ApproveUser(pending.String(), nil, admin.String()); err == nil {
		t.Fatal("approval without department succeeded")
	}
	var pendingStatus string
	tx.Table("users").Select("status").Where("id=?", pending).Scan(&pendingStatus)
	if pendingStatus != "pending_approval" {
		t.Fatal("failed approval changed account status")
	}
	if err := dao.ApproveUser(pending.String(), []uuid.UUID{root.ID, uuid.New()}, admin.String()); err == nil {
		t.Fatal("invalid department accepted")
	}
	tx.Model(&Membership{}).Where("user_id=?", pending).Count(&count)
	if count != 0 {
		t.Fatal("failed approval left partial memberships")
	}
	disabled := create("已停用部门", "", nil)
	disableInput := input(disabled)
	disableInput.Status = "disabled"
	request("PUT", "/departments/"+disabled.ID.String(), disableInput, token, 200)
	if err := dao.ApproveUser(pending.String(), []uuid.UUID{disabled.ID}, admin.String()); err == nil {
		t.Fatal("disabled department accepted")
	}
	if err := dao.ApproveUser(pending.String(), []uuid.UUID{root.ID, sibling.ID, root.ID}, admin.String()); err != nil {
		t.Fatal(err)
	}
	tx.Model(&Membership{}).Where("user_id=?", pending).Count(&count)
	if count != 2 {
		t.Fatalf("approval multi-department assignment: %d", count)
	}
	tx.Table("users").Select("status").Where("id=?", pending).Scan(&pendingStatus)
	if pendingStatus != "active" {
		t.Fatal("approval did not activate user")
	}
	if err := dao.ApproveUser(pending.String(), []uuid.UUID{root.ID}, admin.String()); err == nil {
		t.Fatal("repeated approval accepted")
	}
	if err := dao.UnlockUser(admin.String()); err == nil {
		t.Fatal("non-locked unassigned user unlocked")
	}
	if err := dao.RejectUser(pending.String()); err == nil {
		t.Fatal("approved user rejected through stale approval")
	}
	rollbackUser := uuid.New()
	if err := tx.Exec("INSERT INTO users (id,username,email,status,role) VALUES (?, 'rollback', 'rollback@example.test','pending_approval','user')", rollbackUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("ALTER TABLE audit_logs ADD CONSTRAINT approval_audit_failure CHECK (action <> 'approve_user') NOT VALID").Error; err != nil {
		t.Fatal(err)
	}
	if err := dao.ApproveUser(rollbackUser.String(), []uuid.UUID{root.ID}, admin.String()); err == nil {
		t.Fatal("approval succeeded without an audit")
	}
	tx.Model(&Membership{}).Where("user_id=?", rollbackUser).Count(&count)
	if count != 0 {
		t.Fatal("audit failure left memberships")
	}
	tx.Table("users").Select("status").Where("id=?", rollbackUser).Scan(&pendingStatus)
	if pendingStatus != "pending_approval" {
		t.Fatal("audit failure activated account")
	}
	if err := tx.Exec("ALTER TABLE audit_logs DROP CONSTRAINT approval_audit_failure").Error; err != nil {
		t.Fatal(err)
	}
	var audits int64
	// User-centric replacement is atomic, preserves retained manager roles and detects stale edits.
	sibling = read(sibling.ID)
	request("PUT", "/departments/"+sibling.ID.String()+"/members/"+member.String(), gin.H{"is_manager": true, "version": sibling.Version}, token, 200)
	userURL := "/users/" + member.String() + "/departments"
	request("GET", userURL, nil, memberToken, 403)
	var snapshot userDepartmentSnapshot
	json.Unmarshal(request("GET", userURL, nil, token, 200), &snapshot)
	if len(snapshot.Departments) != 1 || !snapshot.Departments[0].IsManager {
		t.Fatal("user department snapshot missing manager")
	}
	request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{}, "expected_version": snapshot.Version}, token, 400)
	request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{child.ID}, "expected_version": "stale"}, token, 409)
	request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{child.ID}, "expected_version": snapshot.Version}, token, 409)
	request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{disabled.ID}, "expected_version": snapshot.Version, "confirm_manager_removal": true}, token, 409)
	var after userDepartmentSnapshot
	json.Unmarshal(request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{sibling.ID, child.ID, child.ID}, "expected_version": snapshot.Version}, token, 200), &after)
	if len(after.Departments) != 2 || after.Version == snapshot.Version {
		t.Fatal("replacement or version failed")
	}
	for _, d := range after.Departments {
		if d.ID == sibling.ID && !d.IsManager {
			t.Fatal("retained manager role lost")
		}
		if d.ID == child.ID && d.IsManager {
			t.Fatal("new member acquired manager role")
		}
	}
	json.Unmarshal(request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{child.ID}, "expected_version": after.Version, "confirm_manager_removal": true}, token, 200), &after)
	if len(after.Departments) != 1 || after.Departments[0].ID != child.ID || after.Departments[0].IsManager {
		t.Fatal("confirmed manager removal failed")
	}
	request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{root.ID}, "expected_version": snapshot.Version, "confirm_manager_removal": true}, token, 409)
	if err := tx.Exec("ALTER TABLE audit_logs ADD CONSTRAINT transfer_audit_failure CHECK (action <> 'update_user_departments') NOT VALID").Error; err != nil {
		t.Fatal(err)
	}
	request("PUT", userURL, gin.H{"department_ids": []uuid.UUID{root.ID}, "expected_version": after.Version}, token, 500)
	var unchanged userDepartmentSnapshot
	json.Unmarshal(request("GET", userURL, nil, token, 200), &unchanged)
	if unchanged.Version != after.Version || len(unchanged.Departments) != 1 || unchanged.Departments[0].ID != child.ID {
		t.Fatal("audit failure left partial user transfer")
	}
	if err := tx.Exec("ALTER TABLE audit_logs DROP CONSTRAINT transfer_audit_failure").Error; err != nil {
		t.Fatal(err)
	}
	tx.Model(&modelsAudit{}).Count(&audits)
	if audits < 7 {
		t.Fatalf("audit records missing: %d", audits)
	}
	var role string
	tx.Table("users").Select("role").Where("id=?", member).Scan(&role)
	if role != "user" {
		t.Fatal("organization changed global user role")
	}
}

type modelsAudit struct{}

func (modelsAudit) TableName() string { return "audit_logs" }
