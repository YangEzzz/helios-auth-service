package organization

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/middleware"
	"helios-auth-service/internal/models"
	"helios-auth-service/internal/session"
)

type Handler struct{ DB *gorm.DB }
type problem struct {
	status  int
	message string
}

func (p *problem) Error() string { return p.message }
func invalid(s string) error     { return &problem{400, s} }
func conflict(s string) error    { return &problem{409, s} }
func respond(c *gin.Context, data any, err error) {
	if err != nil {
		status, message := 500, "部门操作失败，请稍后重试"
		var p *problem
		var pg interface{ SQLState() string }
		switch {
		case errors.As(err, &p):
			status, message = p.status, p.message
		case errors.Is(err, gorm.ErrRecordNotFound):
			status, message = 404, "部门或成员不存在"
		case errors.As(err, &pg) && pg.SQLState() == "23505":
			status, message = 409, "同级部门名称已存在或内部标识冲突，请重试"
		}
		c.JSON(status, gin.H{"code": status, "message": message})
		return
	}
	c.JSON(200, gin.H{"code": 200, "message": "操作成功", "data": data})
}

func Register(r *gin.RouterGroup, db *gorm.DB, secret, origin string) {
	h := &Handler{db}
	g := r.Group("/org", middleware.AuthMiddleware(secret, db))
	g.GET("/me", h.Me)
	g.Use(middleware.RequireRoles(db, constant.UserRoleAdmin, constant.UserRoleSuperAdmin))
	g.Use(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			if _, err := c.Cookie(session.CookieName); err == nil && !session.Guard(c, session.Origin(origin)) {
				return
			}
		}
		c.Next()
	})
	g.GET("/departments", h.List)
	g.POST("/departments", h.Create)
	g.PUT("/departments/:id", h.Update)
	g.GET("/departments/:id/members", h.Members)
	g.PUT("/departments/:id/members/:userId", h.SaveMember)
	g.DELETE("/departments/:id/members/:userId", h.RemoveMember)
	g.GET("/users/:userId/departments", h.UserDepartments)
	g.PUT("/users/:userId/departments", h.ReplaceUserDepartments)
}

func pathID(c *gin.Context, key string) (uuid.UUID, error) {
	id, e := uuid.Parse(c.Param(key))
	if e != nil || id == uuid.Nil {
		return uuid.Nil, invalid("无效的标识")
	}
	return id, nil
}
func page(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if p < 1 || p > 100000 {
		p = 1
	}
	return p, 20
}

func (h *Handler) List(c *gin.Context) {
	rows := []Department{}
	err := h.DB.WithContext(c.Request.Context()).Raw(`SELECT d.*,
 (SELECT count(*) FROM department_memberships m WHERE m.department_id=d.id) member_count,
 (SELECT count(*) FROM department_memberships m WHERE m.department_id=d.id AND m.is_manager) manager_count
 FROM departments d ORDER BY sort_order, name, id`).Scan(&rows).Error
	respond(c, rows, err)
}
func (h *Handler) Me(c *gin.Context) {
	rows := []Department{}
	err := h.DB.WithContext(c.Request.Context()).Table("departments d").Select("d.*").Joins("JOIN department_memberships m ON m.department_id=d.id").Where("m.user_id=?", c.GetString("user_id")).Order("d.sort_order,d.name,d.id").Scan(&rows).Error
	respond(c, rows, err)
}

type departmentInput struct {
	ParentID  *uuid.UUID `json:"parent_id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	SortOrder int        `json:"sort_order"`
	Status    string     `json:"status"`
	Version   int        `json:"version"`
}

func (v *departmentInput) validate() error {
	v.Name = strings.TrimSpace(v.Name)
	v.Code = strings.TrimSpace(v.Code)
	if utf8.RuneCountInString(v.Name) < 1 || utf8.RuneCountInString(v.Name) > 100 {
		return invalid("部门名称应为 1–100 个字符")
	}
	if len(v.Code) > 64 {
		return invalid("部门编码应为 1–64 个英文字母、数字、下划线或短横线")
	}
	for _, r := range v.Code {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return invalid("部门编码仅支持英文字母、数字、下划线和短横线")
		}
	}
	if v.SortOrder < 0 || v.SortOrder > 1000000 {
		return invalid("排序应为 0–1000000 的整数")
	}
	if v.Status != "active" && v.Status != "disabled" {
		return invalid("请选择有效的部门状态")
	}
	if v.ParentID != nil && *v.ParentID == uuid.Nil {
		return invalid("请选择有效的上级部门")
	}
	return nil
}

// All organization mutations serialize, including tree moves and member changes.
// This prevents concurrent reparent operations forming a cycle or adding members while disabling.
func (h *Handler) mutate(c *gin.Context, fn func(*gorm.DB) (any, error)) {
	var result any
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SELECT pg_advisory_xact_lock(72410831)").Error; e != nil {
			return e
		}
		var e error
		result, e = fn(tx)
		return e
	})
	respond(c, result, err)
}
func audit(tx *gorm.DB, c *gin.Context, action string, id uuid.UUID, before, after any) error {
	actor, e := uuid.Parse(c.GetString("user_id"))
	if e != nil {
		return e
	}
	details, e := json.Marshal(gin.H{"before": before, "after": after})
	if e != nil {
		return e
	}
	return tx.Create(models.NewAuditLog(&actor, action, "department:"+id.String(), string(details), c.ClientIP())).Error
}
func checkParent(tx *gorm.DB, id uuid.UUID, parent *uuid.UUID, active bool) error {
	seen := map[uuid.UUID]bool{id: true}
	for parent != nil {
		if seen[*parent] {
			return invalid("部门不能移动到自身或其下级部门")
		}
		seen[*parent] = true
		var d Department
		if e := tx.First(&d, "id=?", *parent).Error; e != nil {
			return e
		}
		if active && d.Status != "active" {
			return invalid("启用部门不能放在已停用部门下")
		}
		parent = d.ParentID
	}
	return nil
}
func (h *Handler) Create(c *gin.Context) {
	var v departmentInput
	if c.ShouldBindJSON(&v) != nil {
		respond(c, nil, invalid("请检查部门信息"))
		return
	}
	if e := v.validate(); e != nil {
		respond(c, nil, e)
		return
	}
	h.mutate(c, func(tx *gorm.DB) (any, error) {
		if v.Code == "" {
			v.Code = uuid.NewString()
		}
		d := Department{ID: uuid.New(), ParentID: v.ParentID, Code: v.Code, Name: v.Name, SortOrder: v.SortOrder, Status: v.Status, Version: 1}
		if e := checkParent(tx, d.ID, d.ParentID, d.Status == "active"); e != nil {
			return nil, e
		}
		if e := tx.Create(&d).Error; e != nil {
			return nil, e
		}
		return d, audit(tx, c, "create_department", d.ID, nil, d)
	})
}
func (h *Handler) Update(c *gin.Context) {
	id, e := pathID(c, "id")
	if e != nil {
		respond(c, nil, e)
		return
	}
	var v departmentInput
	if c.ShouldBindJSON(&v) != nil {
		respond(c, nil, invalid("请检查部门信息"))
		return
	}
	if e := v.validate(); e != nil {
		respond(c, nil, e)
		return
	}
	h.mutate(c, func(tx *gorm.DB) (any, error) {
		var d Department
		if e := tx.First(&d, "id=?", id).Error; e != nil {
			return nil, e
		}
		if d.Version != v.Version {
			return nil, conflict("部门已被修改，请刷新后重新编辑")
		}
		if e := checkParent(tx, id, v.ParentID, v.Status == "active"); e != nil {
			return nil, e
		}
		if v.Status == "disabled" {
			var children, members int64
			if e := tx.Model(&Department{}).Where("parent_id=? AND status='active'", id).Count(&children).Error; e != nil {
				return nil, e
			}
			if e := tx.Model(&Membership{}).Where("department_id=?", id).Count(&members).Error; e != nil {
				return nil, e
			}
			if children > 0 || members > 0 {
				return nil, conflict("请先移出所有成员，并停用或移走启用的下级部门")
			}
		}
		before := d
		d.ParentID = v.ParentID
		if v.Code != "" {
			d.Code = v.Code
		}
		d.Name = v.Name
		d.SortOrder = v.SortOrder
		d.Status = v.Status
		d.Version++
		if e := tx.Save(&d).Error; e != nil {
			return nil, e
		}
		return d, audit(tx, c, "update_department", id, before, d)
	})
}
func (h *Handler) Members(c *gin.Context) {
	id, e := pathID(c, "id")
	if e != nil {
		respond(c, nil, e)
		return
	}
	var d Department
	db := h.DB.WithContext(c.Request.Context())
	if e := db.First(&d, "id=?", id).Error; e != nil {
		respond(c, nil, e)
		return
	}
	p, size := page(c)
	candidates := c.Query("candidates") == "true"
	q := db.Table("users u").Select("u.id user_id,u.username name,u.email,u.status,COALESCE(m.is_manager,false) is_manager").Joins("LEFT JOIN department_memberships m ON m.user_id=u.id AND m.department_id=?", id)
	if candidates {
		q = q.Where("m.user_id IS NULL AND u.status IN ('active','locked')")
	} else {
		q = q.Where("m.user_id IS NOT NULL")
	}
	if s := strings.TrimSpace(c.Query("q")); s != "" {
		q = q.Where("(u.username ILIKE ? OR u.email ILIKE ?)", "%"+s+"%", "%"+s+"%")
	}
	var total int64
	if e := q.Count(&total).Error; e != nil {
		respond(c, nil, e)
		return
	}
	rows := []Member{}
	err := q.Order("u.username,u.id").Offset((p - 1) * size).Limit(size).Scan(&rows).Error
	respond(c, gin.H{"members": rows, "total": total, "page": p, "page_size": size}, err)
}
func (h *Handler) SaveMember(c *gin.Context)   { h.memberMutation(c, false) }
func (h *Handler) RemoveMember(c *gin.Context) { h.memberMutation(c, true) }
func (h *Handler) memberMutation(c *gin.Context, remove bool) {
	id, e := pathID(c, "id")
	if e != nil {
		respond(c, nil, e)
		return
	}
	uid, e := pathID(c, "userId")
	if e != nil {
		respond(c, nil, e)
		return
	}
	var v struct {
		IsManager bool `json:"is_manager"`
		Version   int  `json:"version"`
	}
	if c.ShouldBindJSON(&v) != nil {
		respond(c, nil, invalid("请检查成员信息"))
		return
	}
	h.mutate(c, func(tx *gorm.DB) (any, error) {
		var d Department
		if e := tx.First(&d, "id=?", id).Error; e != nil {
			return nil, e
		}
		if d.Version != v.Version {
			return nil, conflict("部门成员已变化，请刷新后重试")
		}
		var before *Membership
		var old Membership
		e := tx.Where("department_id=? AND user_id=?", id, uid).First(&old).Error
		if e == nil {
			before = &old
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, e
		}
		action := "save_department_member"
		var after *Membership
		if remove {
			if before == nil {
				return nil, gorm.ErrRecordNotFound
			}
			var memberUser models.User
			if e := tx.First(&memberUser, "id=?", uid).Error; e != nil {
				return nil, e
			}
			if memberUser.Status == constant.UserStatusActive || memberUser.Status == constant.UserStatusLocked {
				var count int64
				if e := tx.Model(&Membership{}).Where("user_id=?", uid).Count(&count).Error; e != nil {
					return nil, e
				}
				if count <= 1 {
					return nil, conflict("用户必须保留至少一个部门，请先加入新部门再移出当前部门")
				}
			}
			if e := tx.Delete(&Membership{}, "department_id=? AND user_id=?", id, uid).Error; e != nil {
				return nil, e
			}
			action = "remove_department_member"
		} else {
			if d.Status != "active" {
				return nil, conflict("已停用部门不能加入成员或设置负责人")
			}
			var user models.User
			if e := tx.First(&user, "id=?", uid).Error; e != nil {
				return nil, e
			}
			if before == nil && user.Status != constant.UserStatusActive && user.Status != constant.UserStatusLocked {
				return nil, invalid("只能添加已审批通过的用户")
			}
			m := Membership{DepartmentID: id, UserID: uid, IsManager: v.IsManager}
			if before != nil {
				m.CreatedAt = before.CreatedAt
			}
			if e := tx.Save(&m).Error; e != nil {
				return nil, e
			}
			after = &m
		}
		if e := tx.Model(&d).Updates(map[string]any{"version": d.Version + 1, "updated_at": gorm.Expr("now()")}).Error; e != nil {
			return nil, e
		}
		return gin.H{"version": d.Version}, audit(tx, c, action, id, before, after)
	})
}
