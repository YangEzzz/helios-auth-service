package organization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"
)

type userDepartmentSnapshot struct {
	Departments []models.UserDepartment `json:"departments"`
	Version     string                  `json:"version"`
}

func membershipSnapshot(tx *gorm.DB, uid uuid.UUID) (userDepartmentSnapshot, []Membership, error) {
	memberships := []Membership{}
	if err := tx.Where("user_id=?", uid).Order("department_id").Find(&memberships).Error; err != nil {
		return userDepartmentSnapshot{}, nil, err
	}
	raw, _ := json.Marshal(memberships)
	sum := sha256.Sum256(raw)
	result := userDepartmentSnapshot{Departments: []models.UserDepartment{}, Version: hex.EncodeToString(sum[:])}
	err := tx.Table("departments d").Select("d.id,d.name,d.status,m.is_manager").Joins("JOIN department_memberships m ON m.department_id=d.id").Where("m.user_id=?", uid).Order("d.sort_order,d.name,d.id").Scan(&result.Departments).Error
	return result, memberships, err
}

func (h *Handler) UserDepartments(c *gin.Context) {
	uid, err := pathID(c, "userId")
	if err != nil {
		respond(c, nil, err)
		return
	}
	h.mutate(c, func(tx *gorm.DB) (any, error) {
		var user models.User
		if err := tx.First(&user, "id=?", uid).Error; err != nil {
			return nil, err
		}
		result, _, err := membershipSnapshot(tx, uid)
		return result, err
	})
}

func (h *Handler) ReplaceUserDepartments(c *gin.Context) {
	uid, err := pathID(c, "userId")
	if err != nil {
		respond(c, nil, err)
		return
	}
	var input struct {
		DepartmentIDs         []uuid.UUID `json:"department_ids" binding:"required,min=1,max=100"`
		ExpectedVersion       string      `json:"expected_version" binding:"required"`
		ConfirmManagerRemoval bool        `json:"confirm_manager_removal"`
	}
	if c.ShouldBindJSON(&input) != nil {
		respond(c, nil, invalid("请选择至少一个部门，并携带当前成员版本"))
		return
	}
	wanted := map[uuid.UUID]bool{}
	for _, id := range input.DepartmentIDs {
		if id == uuid.Nil {
			respond(c, nil, invalid("请选择有效部门"))
			return
		}
		wanted[id] = true
	}
	h.mutate(c, func(tx *gorm.DB) (any, error) {
		var user models.User
		if err := tx.First(&user, "id=?", uid).Error; err != nil {
			return nil, err
		}
		if user.Status != constant.UserStatusActive && user.Status != constant.UserStatusLocked {
			return nil, invalid("仅已通过审批的账户可以修改部门")
		}
		before, current, err := membershipSnapshot(tx, uid)
		if err != nil {
			return nil, err
		}
		if input.ExpectedVersion != before.Version {
			return nil, conflict("用户部门或负责人身份已变化，请刷新后重新选择")
		}
		existing := map[uuid.UUID]Membership{}
		for _, m := range current {
			existing[m.DepartmentID] = m
		}
		for id := range wanted {
			var count int64
			if err := tx.Model(&Department{}).Where("id=? AND status='active'", id).Count(&count).Error; err != nil {
				return nil, err
			}
			if count != 1 {
				return nil, conflict("所选部门已停用或不存在，请刷新部门列表")
			}
		}
		changed := []uuid.UUID{}
		for _, m := range current {
			if wanted[m.DepartmentID] {
				continue
			}
			if m.IsManager && !input.ConfirmManagerRemoval {
				return nil, conflict("移出负责人所在部门会取消其负责人身份，请确认后再保存")
			}
			if err := tx.Delete(&Membership{}, "department_id=? AND user_id=?", m.DepartmentID, uid).Error; err != nil {
				return nil, err
			}
			changed = append(changed, m.DepartmentID)
		}
		for id := range wanted {
			if _, ok := existing[id]; ok {
				continue
			}
			if err := tx.Create(&Membership{DepartmentID: id, UserID: uid}).Error; err != nil {
				return nil, err
			}
			changed = append(changed, id)
		}
		if len(changed) == 0 {
			return before, nil
		}
		if err := tx.Model(&Department{}).Where("id IN ?", changed).Updates(map[string]any{"version": gorm.Expr("version+1"), "updated_at": gorm.Expr("now()")}).Error; err != nil {
			return nil, err
		}
		after, _, err := membershipSnapshot(tx, uid)
		if err != nil {
			return nil, err
		}
		actor, err := uuid.Parse(c.GetString("user_id"))
		if err != nil {
			return nil, err
		}
		details, _ := json.Marshal(gin.H{"before": before.Departments, "after": after.Departments})
		if err := tx.Create(models.NewAuditLog(&actor, "update_user_departments", "user:"+uid.String(), string(details), c.ClientIP())).Error; err != nil {
			return nil, err
		}
		return after, nil
	})
}
