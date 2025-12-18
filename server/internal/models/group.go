package models

import "time"

type GroupStatus string

const (
	GroupStatusConnected GroupStatus = "Connected"
	GroupStatusUnbound   GroupStatus = "Unbound"
	GroupStatusInactive  GroupStatus = "Inactive"
)

type Group struct {
	ID                string      `json:"id" gorm:"primaryKey"` // Telegram Chat ID
	Title             string      `json:"title"`
	Status            GroupStatus `json:"status" gorm:"default:'Unbound'"`
	DatabaseID        *string     `json:"database_id"`
	NotionAccessToken string      `json:"-"`             // Encrypted
	DatabaseName      string      `json:"database_name"` // Cached name for UI
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type GroupRole string

const (
	GroupRoleAdmin  GroupRole = "Admin"
	GroupRoleMember GroupRole = "Member"
)

type UserGroup struct {
	UserID    string    `json:"user_id" gorm:"primaryKey"`
	GroupID   string    `json:"group_id" gorm:"primaryKey"`
	Role      GroupRole `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
