package sso

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"helios-auth-service/internal/audit"
	"helios-auth-service/internal/auth"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/project"
	"helios-auth-service/internal/session"
)

type Handler struct {
	DB       *gorm.DB
	Frontend string
	Secure   bool
	Auth     auth.Service
	Project  project.Service
	Audit    audit.Service
}

func Register(r *gin.Engine, db *gorm.DB, secret, frontend string, secure bool) {
	a := audit.NewService(audit.NewDao(db))
	h := &Handler{db, strings.TrimRight(frontend, "/"), secure, auth.NewService(auth.NewDao(db), secret), project.NewService(project.NewDao(db), a), a}
	g := r.Group("/api/sso")
	g.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 65536)
		c.Header("Cache-Control", "no-store")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	})
	g.GET("/session", h.Session)
	g.GET("/authorize", h.Authorize)
	g.GET("/request", h.RequestInfo)
	g.POST("/login", LoginLimit(), h.Login)
	g.POST("/continue", h.Continue)
	g.POST("/cancel", h.Cancel)
	g.POST("/logout", h.Logout)
	g.POST("/exchange", h.Exchange)
	g.POST("/identity", h.Identity)
	admin := r.Group("/api/v1/sso")
	admin.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 65536)
		c.Next()
	})
	admin.Use(middleware.AuthMiddleware(secret, db), middleware.RequireRoles(db, constant.UserRoleAdmin, constant.UserRoleSuperAdmin))
	admin.GET("/clients", h.ListClients)
	admin.POST("/clients", h.SaveClient)
	admin.POST("/clients/:id/rotate", h.Rotate)
}
func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "message": msg})
}
func success(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 200, "data": data, "message": "操作成功"})
}
func (h *Handler) csrf(c *gin.Context) bool { return session.Guard(c, session.Origin(h.Frontend)) }
func profile(u *models.User) gin.H {
	return gin.H{"id": u.ID, "name": u.Username, "email": u.Email, "avatar": u.Avatar, "role": u.Role}
}
func (h *Handler) Session(c *gin.Context) {
	csrf, err := session.CSRF(c, h.Secure)
	if err != nil {
		fail(c, 500, "无法初始化登录")
		return
	}
	u, err := session.User(c, h.DB)
	if err != nil {
		success(c, gin.H{"user": nil, "csrf_token": csrf})
		return
	}
	success(c, gin.H{"user": profile(u), "csrf_token": csrf})
}
func validRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))
}
func (h *Handler) client(id, redirect string) (*Client, error) {
	var cl Client
	if err := h.DB.First(&cl, "id = ? AND enabled = ?", id, true).Error; err != nil {
		return nil, err
	}
	for _, uri := range cl.RedirectURIs {
		if uri == redirect {
			return &cl, nil
		}
	}
	return nil, errors.New("unregistered redirect")
}
func (h *Handler) Authorize(c *gin.Context) {
	cl, err := h.client(c.Query("client_id"), c.Query("redirect_uri"))
	if err != nil {
		fail(c, 400, "接入应用或回调地址无效")
		return
	}
	state, challenge := c.Query("state"), c.Query("code_challenge")
	if len(state) < 32 || len(state) > 128 || len(challenge) != 43 || c.Query("code_challenge_method") != "S256" {
		fail(c, 400, "授权参数无效")
		return
	}
	browser, _ := c.Cookie("helios_browser")
	if len(browser) != 43 {
		browser, err = session.Random()
		if err != nil {
			fail(c, 500, "无法初始化登录")
			return
		}
		session.Cookie(c, "helios_browser", browser, 8*3600, h.Secure, true)
	}
	id, err := session.Random()
	if err != nil {
		fail(c, 500, "无法初始化登录")
		return
	}
	req := Request{ID: session.Hash(id), BrowserHash: session.Hash(browser), ClientID: cl.ID, RedirectURI: c.Query("redirect_uri"), State: state, Challenge: challenge, ExpiresAt: time.Now().Add(10 * time.Minute)}
	if err = h.DB.Create(&req).Error; err != nil {
		fail(c, 500, "无法初始化登录")
		return
	}
	c.Redirect(302, h.Frontend+"/sso/login?request="+url.QueryEscape(id))
}
func (h *Handler) request(c *gin.Context, id string) (*Request, error) {
	browser, _ := c.Cookie("helios_browser")
	var req Request
	err := h.DB.Where("id = ? AND browser_hash = ? AND expires_at > ? AND used_at IS NULL", session.Hash(id), session.Hash(browser), time.Now()).First(&req).Error
	return &req, err
}
func (h *Handler) RequestInfo(c *gin.Context) {
	req, err := h.request(c, c.Query("id"))
	if err != nil {
		fail(c, 400, "登录请求已失效，请返回应用重新登录")
		return
	}
	cl, err := h.client(req.ClientID, req.RedirectURI)
	if err != nil {
		fail(c, 400, "接入应用已停用")
		return
	}
	csrf, err := session.CSRF(c, h.Secure)
	if err != nil {
		fail(c, 500, "无法初始化登录")
		return
	}
	u, err := session.User(c, h.DB)
	data := gin.H{"app_name": cl.Name, "csrf_token": csrf, "user": nil}
	if err == nil {
		data["user"] = profile(u)
	}
	success(c, data)
}
func (h *Handler) Login(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	var in struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		RequestID string `json:"request_id"`
	}
	if c.ShouldBindJSON(&in) != nil || len(in.Email) > 255 || len(in.Password) > 128 {
		fail(c, 400, "请填写邮箱和密码")
		return
	}
	if in.RequestID != "" {
		if _, err := h.request(c, in.RequestID); err != nil {
			fail(c, 400, "登录请求已失效，请重新登录")
			return
		}
	}
	u, _, err := h.Auth.Login(c.Request.Context(), in.Email, in.Password)
	if err != nil {
		fail(c, 401, "邮箱、密码不正确，或账号暂不可用")
		return
	}
	if err = session.Create(c, h.DB, u.ID, h.Secure); err != nil {
		fail(c, 500, "无法建立登录会话")
		return
	}
	_ = h.Audit.LogAction(c.Request.Context(), &u.ID, constant.ActionUserLogin, "user:"+u.ID.String(), "SSO login", c.ClientIP())
	success(c, gin.H{"user": profile(u)})
}
func (h *Handler) Continue(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	var in struct {
		RequestID string `json:"request_id"`
	}
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "请求无效")
		return
	}
	req, err := h.request(c, in.RequestID)
	if err != nil {
		fail(c, 400, "登录请求已失效，请重新登录")
		return
	}
	cl, err := h.client(req.ClientID, req.RedirectURI)
	if err != nil {
		fail(c, 400, "接入应用已停用")
		return
	}
	u, err := session.User(c, h.DB)
	if err != nil {
		fail(c, 401, "登录已失效，请重新登录")
		return
	}
	var p models.Project
	if h.DB.First(&p, "id = ?", cl.ProjectID).Error != nil {
		fail(c, 403, "项目不可用")
		return
	}
	role, err := h.Project.VerifyUserProjectRole(c.Request.Context(), u.ID, p.ProjectIDString)
	if err != nil {
		fail(c, 503, "暂时无法校验项目权限，请重试")
		return
	}
	if role == "" {
		fail(c, 403, "你尚未获得此应用的访问权限，请联系管理员")
		return
	}
	code, err := session.Random()
	if err != nil {
		fail(c, 500, "无法完成登录")
		return
	}
	now := time.Now()
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Request{}).Where("id = ? AND used_at IS NULL AND expires_at > ?", req.ID, now).Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("request consumed")
		}
		return tx.Create(&Code{ID: session.Hash(code), UserID: u.ID, ClientID: cl.ID, RedirectURI: req.RedirectURI, Challenge: req.Challenge, ExpiresAt: now.Add(time.Minute)}).Error
	})
	if err != nil {
		fail(c, 400, "登录请求已完成或失效，请重新登录")
		return
	}
	target, _ := url.Parse(req.RedirectURI)
	q := target.Query()
	q.Set("code", code)
	q.Set("state", req.State)
	target.RawQuery = q.Encode()
	success(c, gin.H{"redirect_url": target.String()})
}
func (h *Handler) Cancel(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	var in struct {
		RequestID string `json:"request_id"`
	}
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "请求无效")
		return
	}
	req, err := h.request(c, in.RequestID)
	if err != nil {
		fail(c, 400, "登录请求已失效")
		return
	}
	result := h.DB.Model(&Request{}).Where("id = ? AND used_at IS NULL", req.ID).Update("used_at", time.Now())
	if result.Error != nil || result.RowsAffected != 1 {
		fail(c, 400, "登录请求已失效")
		return
	}
	target, _ := url.Parse(req.RedirectURI)
	q := target.Query()
	q.Set("error", "access_denied")
	q.Set("state", req.State)
	target.RawQuery = q.Encode()
	success(c, gin.H{"redirect_url": target.String()})
}
func (h *Handler) Logout(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	if session.Revoke(c, h.DB, h.Secure) != nil {
		fail(c, 500, "退出失败，请重试")
		return
	}
	success(c, gin.H{})
}
func (h *Handler) authenticate(c *gin.Context) (*Client, error) {
	id, secret, ok := c.Request.BasicAuth()
	if !ok {
		return nil, errors.New("missing client credentials")
	}
	var cl Client
	if h.DB.First(&cl, "id = ? AND enabled = ?", id, true).Error != nil || !session.Equal(cl.SecretHash, session.Hash(secret)) {
		return nil, errors.New("invalid client")
	}
	return &cl, nil
}

type Identity struct {
	User      gin.H  `json:"user"`
	ProjectID string `json:"project_id_string"`
	Role      string `json:"role_in_project"`
}

func identity(tx *gorm.DB, cl *Client, userID uuid.UUID) (*Identity, error) {
	var u models.User
	var p models.Project
	var m models.ProjectMembership
	if err := tx.First(&u, "id = ? AND status = ?", userID, constant.UserStatusActive).Error; err != nil {
		return nil, err
	}
	if err := tx.First(&p, "id = ?", cl.ProjectID).Error; err != nil {
		return nil, err
	}
	if err := tx.First(&m, "user_id = ? AND project_id = ?", u.ID, p.ID).Error; err != nil {
		return nil, err
	}
	if m.RoleInProject == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &Identity{profile(&u), p.ProjectIDString, m.RoleInProject}, nil
}
func (h *Handler) Exchange(c *gin.Context) {
	cl, err := h.authenticate(c)
	if err != nil {
		fail(c, 401, "接入应用认证失败")
		return
	}
	var in struct {
		Code        string `json:"code"`
		RedirectURI string `json:"redirect_uri"`
		Verifier    string `json:"code_verifier"`
	}
	if c.ShouldBindJSON(&in) != nil || len(in.Code) != 43 || len(in.Verifier) < 43 || len(in.Verifier) > 128 {
		fail(c, 400, "授权码无效")
		return
	}
	if _, err = h.client(cl.ID, in.RedirectURI); err != nil {
		fail(c, 400, "回调地址无效")
		return
	}
	var info *Identity
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var code Code
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&code, "id = ?", session.Hash(in.Code)).Error; err != nil {
			return err
		}
		if code.UsedAt != nil || !code.ExpiresAt.After(time.Now()) || code.ClientID != cl.ID || code.RedirectURI != in.RedirectURI || !session.Equal(code.Challenge, session.Challenge(in.Verifier)) {
			return errors.New("invalid code")
		}
		info, err = identity(tx, cl, code.UserID)
		if err != nil {
			return err
		}
		return tx.Model(&code).Update("used_at", time.Now()).Error
	})
	if err != nil {
		fail(c, 400, "授权码已失效、已使用或权限不可用")
		return
	}
	uid := info.User["id"].(uuid.UUID)
	_ = h.Audit.LogAction(c.Request.Context(), &uid, constant.ActionUserExternalLogin, "project:"+info.ProjectID, "SSO code exchange", c.ClientIP())
	success(c, info)
}
func (h *Handler) Identity(c *gin.Context) {
	cl, err := h.authenticate(c)
	if err != nil {
		fail(c, 401, "接入应用认证失败")
		return
	}
	var in struct {
		UserID string `json:"user_id"`
	}
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "请求无效")
		return
	}
	id, err := uuid.Parse(in.UserID)
	if err != nil {
		fail(c, 400, "用户无效")
		return
	}
	info, err := identity(h.DB, cl, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, 403, "账号或项目权限已失效")
		} else {
			fail(c, 503, "暂时无法校验访问权限，请稍后重试")
		}
		return
	}
	success(c, info)
}
func (h *Handler) ListClients(c *gin.Context) {
	projectID, err := uuid.Parse(c.Query("project_id"))
	if err != nil {
		fail(c, 400, "项目无效")
		return
	}
	list := []Client{}
	if h.DB.Where("project_id = ?", projectID).Order("created_at DESC").Limit(100).Find(&list).Error != nil {
		fail(c, 500, "无法加载接入配置")
		return
	}
	success(c, gin.H{"clients": list})
}
func (h *Handler) SaveClient(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	var in struct {
		ID           string    `json:"client_id"`
		ProjectID    uuid.UUID `json:"project_id"`
		Name         string    `json:"name"`
		RedirectURIs []string  `json:"redirect_uris"`
		Enabled      bool      `json:"enabled"`
	}
	if c.ShouldBindJSON(&in) != nil || len(in.ID) < 3 || len(in.ID) > 100 || strings.TrimSpace(in.Name) == "" || len(in.Name) > 255 || len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 10 {
		fail(c, 400, "请填写应用标识、名称和回调地址")
		return
	}
	for _, ch := range in.ID {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			fail(c, 400, "应用标识仅支持字母、数字、横线和下划线")
			return
		}
	}
	for _, uri := range in.RedirectURIs {
		if len(uri) > 2048 || !validRedirect(uri) {
			fail(c, 400, "回调地址须使用 HTTPS；本地开发可使用 localhost HTTP")
			return
		}
	}
	var p models.Project
	if h.DB.First(&p, "id = ?", in.ProjectID).Error != nil {
		fail(c, 400, "项目不存在")
		return
	}
	var cl Client
	secret := ""
	err := h.DB.First(&cl, "id = ?", in.ID).Error
	creating := errors.Is(err, gorm.ErrRecordNotFound)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		secret, err = session.Random()
		if err != nil {
			fail(c, 500, "无法创建密钥")
			return
		}
		cl = Client{ID: in.ID, ProjectID: in.ProjectID, SecretHash: session.Hash(secret)}
	} else if err != nil {
		fail(c, 500, "无法读取应用")
		return
	} else if cl.ProjectID != in.ProjectID {
		fail(c, 400, "应用标识已被其他项目使用")
		return
	}
	cl.Name = strings.TrimSpace(in.Name)
	cl.RedirectURIs = in.RedirectURIs
	cl.Enabled = in.Enabled
	if creating {
		err = h.DB.Create(&cl).Error
	} else {
		err = h.DB.Model(&Client{}).Where("id = ? AND project_id = ?", cl.ID, cl.ProjectID).Select("name", "redirect_uris", "enabled").Updates(&cl).Error
	}
	if err != nil {
		fail(c, 500, "无法保存接入配置")
		return
	}
	op, _ := uuid.Parse(c.GetString("user_id"))
	_ = h.Audit.LogAction(c.Request.Context(), &op, "sso_client_save", "project:"+p.ProjectIDString, "Saved SSO client "+cl.ID, c.ClientIP())
	success(c, gin.H{"client": cl, "client_secret": secret})
}
func (h *Handler) Rotate(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	secret, err := session.Random()
	if err != nil {
		fail(c, 500, "无法生成密钥")
		return
	}
	result := h.DB.Model(&Client{}).Where("id = ?", c.Param("id")).Update("secret_hash", session.Hash(secret))
	if result.Error != nil || result.RowsAffected != 1 {
		fail(c, 404, "接入应用不存在")
		return
	}
	op, _ := uuid.Parse(c.GetString("user_id"))
	_ = h.Audit.LogAction(c.Request.Context(), &op, "sso_client_rotate", "client:"+c.Param("id"), "Rotated SSO client secret", c.ClientIP())
	success(c, gin.H{"client_secret": secret})
}
