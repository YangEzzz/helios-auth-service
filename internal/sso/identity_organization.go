package sso

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"helios-auth-service/internal/models"
)

// OrganizationDepartment is the read-only directory projection for registered applications.
// No other users, member counts, audit details or administrative fields are disclosed.
type OrganizationDepartment struct {
	ID        uuid.UUID  `json:"id"`
	ParentID  *uuid.UUID `json:"parent_id"`
	Name      string     `json:"name"`
	SortOrder int        `json:"sort_order"`
	Status    string     `json:"status"`
}
type OrganizationInfo struct {
	Departments []OrganizationDepartment `json:"departments"`
	Version     string                   `json:"version"`
}

func identity(db *gorm.DB, cl *Client, userID uuid.UUID) (*Identity, error) {
	var result *Identity
	err := db.Transaction(func(tx *gorm.DB) error {
		// Matches all department/member mutations so directory and memberships describe one state.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(72410831)").Error; err != nil {
			return err
		}
		info, err := authorizedIdentity(tx, cl, userID)
		if err != nil {
			return err
		}
		directory := []OrganizationDepartment{}
		if err := tx.Table("departments").Select("id,parent_id,name,sort_order,status").Order("sort_order,name,id").Scan(&directory).Error; err != nil {
			return err
		}
		memberships := []models.UserDepartment{}
		if err := tx.Table("departments d").Select("d.id,d.name,d.status,m.is_manager").Joins("JOIN department_memberships m ON m.department_id=d.id").Where("m.user_id=?", userID).Order("d.sort_order,d.name,d.id").Scan(&memberships).Error; err != nil {
			return err
		}
		raw, err := json.Marshal(directory)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		info.User["departments"] = memberships
		info.Organization = &OrganizationInfo{Departments: directory, Version: hex.EncodeToString(sum[:])}
		result = info
		return nil
	})
	return result, err
}
