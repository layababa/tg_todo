package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a Telegram user in the system
type User struct {
	ID                string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TgID              int64          `gorm:"uniqueIndex;not null" json:"tg_id"`
	TgUsername        string         `gorm:"type:text" json:"tg_username,omitempty"`
	Name              string         `gorm:"type:text;not null" json:"name"`
	PhotoURL          string         `gorm:"type:text" json:"photo_url,omitempty"`
	Avatar            string         `gorm:"type:text" json:"avatar,omitempty"` // Deprecated, use PhotoURL
	Timezone          string         `gorm:"type:text;not null;default:'UTC+0'" json:"timezone"`
	DefaultDatabaseID *string        `gorm:"type:text" json:"default_database_id,omitempty"`
	NotionConnected   bool           `gorm:"not null;default:false" json:"notion_connected"`
	CalendarToken     *string        `gorm:"type:text;uniqueIndex:uni_users_calendar_token" json:"calendar_token,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName overrides the table name
func (User) TableName() string {
	return "users"
}

// UserNotionToken represents encrypted Notion OAuth tokens for a user
type UserNotionToken struct {
	ID              string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          string         `gorm:"type:uuid;not null;index" json:"user_id"`
	User            *User          `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	AccessTokenEnc  string         `gorm:"type:text;not null" json:"-"` // Never expose in JSON
	RefreshTokenEnc string         `gorm:"type:text" json:"-"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
	WorkspaceID     string         `gorm:"type:text;not null" json:"workspace_id"`
	WorkspaceName   string         `gorm:"type:text;not null" json:"workspace_name"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName overrides the table name
func (UserNotionToken) TableName() string {
	return "user_notion_tokens"
}
