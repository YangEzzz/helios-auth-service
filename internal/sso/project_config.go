package sso

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/session"
)

type projectConfig struct {
	ClientID        string   `json:"client_id"`
	Name            string   `json:"name"`
	Configured      bool     `json:"configured"`
	CredentialReady bool     `json:"credential_ready"`
	Enabled         bool     `json:"enabled"`
	MixedEnabled    bool     `json:"mixed_enabled"`
	RedirectURIs    []string `json:"redirect_uris"`
}

func summarizeConfig(p models.Project, clients []Client) projectConfig {
	out := projectConfig{ClientID: p.ProjectIDString, Name: p.ProjectName, Configured: len(clients) > 0, Enabled: true, RedirectURIs: []string{}}
	seen := map[string]bool{}
	for i, cl := range clients {
		if cl.ID == p.ProjectIDString {
			out.CredentialReady = true
		}
		if i == 0 {
			out.Enabled = cl.Enabled
		}
		if cl.Enabled != out.Enabled {
			out.MixedEnabled = true
		}
		for _, uri := range cl.RedirectURIs {
			if !seen[uri] {
				seen[uri] = true
				out.RedirectURIs = append(out.RedirectURIs, uri)
			}
		}
	}
	// Mixed legacy states require an explicit choice rather than silently enabling.
	if out.MixedEnabled {
		out.Enabled = false
	}
	return out
}

func projectAndClients(db *gorm.DB, id string, lock bool) (models.Project, []Client, error) {
	var p models.Project
	var clients []Client
	projectID, err := uuid.Parse(id)
	if err != nil {
		return p, clients, gorm.ErrRecordNotFound
	}
	query := db
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err = query.First(&p, "id = ?", projectID).Error; err != nil {
		return p, clients, err
	}
	err = db.Where("project_id = ?", p.ID).Order("id ASC").Find(&clients).Error
	return p, clients, err
}

func (h *Handler) ProjectConfig(c *gin.Context) {
	p, clients, err := projectAndClients(h.DB, c.Param("project_id"), false)
	if err != nil {
		fail(c, 404, "无法读取项目登录配置")
		return
	}
	success(c, gin.H{"config": summarizeConfig(p, clients)})
}

func (h *Handler) SaveProjectConfig(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	var in struct {
		Enabled      bool     `json:"enabled"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if c.ShouldBindJSON(&in) != nil || len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 10 {
		fail(c, 400, "请填写 1 至 10 个登录返回地址")
		return
	}
	seen := map[string]bool{}
	redirects := []string{}
	for _, raw := range in.RedirectURIs {
		uri := strings.TrimSpace(raw)
		if len(uri) > 2048 || !validRedirect(uri) {
			fail(c, 400, "返回地址须使用 HTTPS；本地可使用 localhost HTTP")
			return
		}
		if !seen[uri] {
			seen[uri] = true
			redirects = append(redirects, uri)
		}
	}
	var p models.Project
	var result projectConfig
	secret := ""
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		p, _, err = projectAndClients(tx, c.Param("project_id"), true)
		if err != nil {
			return err
		}
		var canonical Client
		err = tx.First(&canonical, "id = ?", p.ProjectIDString).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			secret, err = session.Random()
			if err != nil {
				return err
			}
			canonical = Client{ID: p.ProjectIDString, ProjectID: p.ID, Name: p.ProjectName, SecretHash: session.Hash(secret), RedirectURIs: redirects, Enabled: in.Enabled}
			if err = tx.Create(&canonical).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if canonical.ProjectID != p.ID {
			return errors.New("project identifier belongs to another client")
		}
		// Keep old IDs and hashes valid while managing a single project setting.
		if err = tx.Model(&Client{}).Where("project_id = ?", p.ID).Updates(map[string]any{"name": p.ProjectName, "redirect_uris": pq.StringArray(redirects), "enabled": in.Enabled}).Error; err != nil {
			return err
		}
		result = projectConfig{ClientID: p.ProjectIDString, Name: p.ProjectName, Configured: true, CredentialReady: true, Enabled: in.Enabled, RedirectURIs: redirects}
		return nil
	})
	if err != nil {
		fail(c, 409, "无法保存项目登录配置，请检查项目标识是否与历史接入冲突后重试")
		return
	}
	op, _ := uuid.Parse(c.GetString("user_id"))
	_ = h.Audit.LogAction(c.Request.Context(), &op, "sso_project_save", "project:"+p.ProjectIDString, "Saved project sign-in configuration", c.ClientIP())
	success(c, gin.H{"config": result, "client_secret": secret})
}

func (h *Handler) RotateProjectSecret(c *gin.Context) {
	if !h.csrf(c) {
		return
	}
	secret, err := session.Random()
	if err != nil {
		fail(c, 500, "无法生成密钥")
		return
	}
	var p models.Project
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var clients []Client
		var err error
		p, clients, err = projectAndClients(tx, c.Param("project_id"), true)
		if err != nil {
			return err
		}
		if len(clients) == 0 {
			return gorm.ErrRecordNotFound
		}
		if !summarizeConfig(p, clients).CredentialReady {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&Client{}).Where("project_id = ?", p.ID).Update("secret_hash", session.Hash(secret)).Error
	})
	if err != nil {
		fail(c, 409, "请先保存项目登录配置，再更新密钥")
		return
	}
	op, _ := uuid.Parse(c.GetString("user_id"))
	_ = h.Audit.LogAction(c.Request.Context(), &op, "sso_project_rotate", "project:"+p.ProjectIDString, "Rotated project sign-in secret", c.ClientIP())
	success(c, gin.H{"client_secret": secret})
}
