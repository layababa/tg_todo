package models

import (
	"time"
)

// PendingAssignment represents a task assignment for a user who hasn't joined yet
type PendingAssignment struct {
	ID         string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskID     string    `gorm:"type:uuid;not null;index" json:"task_id"`
	TgUsername string    `gorm:"type:text;not null;index" json:"tg_username"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName overrides the table name
func (PendingAssignment) TableName() string {
	return "pending_assignments"
}
