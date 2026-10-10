package sso

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"time"
)

type Client struct {
	ID           string         `json:"client_id" gorm:"primaryKey;size:100"`
	ProjectID    uuid.UUID      `json:"project_id" gorm:"type:uuid;not null;index"`
	Name         string         `json:"name" gorm:"size:255;not null"`
	SecretHash   string         `json:"-" gorm:"size:64;not null"`
	RedirectURIs pq.StringArray `json:"redirect_uris" gorm:"type:text[];not null"`
	Enabled      bool           `json:"enabled"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (Client) TableName() string { return "sso_clients" }

type Request struct {
	ID          string    `gorm:"primaryKey;size:64"`
	BrowserHash string    `gorm:"size:64;not null"`
	ClientID    string    `gorm:"size:100;not null"`
	RedirectURI string    `gorm:"not null"`
	State       string    `gorm:"size:128;not null"`
	Challenge   string    `gorm:"size:43;not null"`
	ExpiresAt   time.Time `gorm:"index;not null"`
	UsedAt      *time.Time
	CreatedAt   time.Time
}

func (Request) TableName() string { return "sso_requests" }

type Code struct {
	ID          string    `gorm:"primaryKey;size:64"`
	UserID      uuid.UUID `gorm:"type:uuid;not null"`
	ClientID    string    `gorm:"size:100;not null"`
	RedirectURI string    `gorm:"not null"`
	Challenge   string    `gorm:"size:43;not null"`
	ExpiresAt   time.Time `gorm:"index;not null"`
	UsedAt      *time.Time
	CreatedAt   time.Time
}

func (Code) TableName() string { return "sso_codes" }
