package task

import "time"

// Task mirrors前端所需字段，供 API 序列化和 Bot 使用。
type Task struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Description   string      `json:"description,omitempty"`
	Status        Status      `json:"status"`
	CreatedAt     time.Time   `json:"createdAt"`
	CreatedBy     Person      `json:"createdBy"`
	Assignees     []Person    `json:"assignees"`
	SourceMessage string      `json:"sourceMessageUrl,omitempty"`
	Permissions   Permissions `json:"permissions"`
}

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
)

type Person struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username,omitempty"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

type Permissions struct {
	CanEdit     bool `json:"canEdit"`
	CanComplete bool `json:"canComplete"`
	CanDelete   bool `json:"canDelete"`
}
