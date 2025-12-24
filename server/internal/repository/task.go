package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusToDo       TaskStatus = "To Do"
	TaskStatusInProgress TaskStatus = "In Progress"
	TaskStatusDone       TaskStatus = "Done"
)

// TaskSyncStatus represents the sync status of a task
type TaskSyncStatus string

const (
	TaskSyncStatusSynced  TaskSyncStatus = "Synced"
	TaskSyncStatusPending TaskSyncStatus = "Pending"
	TaskSyncStatusFailed  TaskSyncStatus = "Failed"
)

// ContextRole represents the role in context snapshot
type ContextRole string

const (
	ContextRoleMe     ContextRole = "me"
	ContextRoleOther  ContextRole = "other"
	ContextRoleSystem ContextRole = "system"
)

// Task represents the tasks table
type Task struct {
	ID              string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NotionPageID    *string        `gorm:"type:text"`
	Title           string         `gorm:"type:text;not null"`
	Description     string         `gorm:"type:text"`
	Status          TaskStatus     `gorm:"type:task_status;default:'To Do';not null"`
	SyncStatus      TaskSyncStatus `gorm:"type:task_sync_status;default:'Pending';not null"`
	GroupID         *string        `gorm:"type:text"` // Telegram Chat ID (matches groups.id)
	DatabaseID      *string        `gorm:"type:text"`
	Topic           string         `gorm:"type:text"`
	DueAt           *time.Time     `gorm:"type:timestamptz"`
	CreatorID       *string        `gorm:"type:uuid"`
	ChatJumpURL     string         `gorm:"type:text"`
	NotionURL       *string        `gorm:"type:text"`
	Archived        bool           `gorm:"default:false"`
	Reminder1hSent  bool           `gorm:"column:reminder_1h_sent;default:false"`
	ReminderDueSent bool           `gorm:"column:reminder_due_sent;default:false"`
	CreatedAt       time.Time      `gorm:"default:now()"`
	UpdatedAt       time.Time      `gorm:"default:now()"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	Assignees []models.User         `gorm:"many2many:task_assignees;"`
	Snapshots []TaskContextSnapshot `gorm:"foreignKey:TaskID"`
	Events    []TaskEvent           `gorm:"foreignKey:TaskID"`
}

// TaskAssignee represents the task_assignees join table
type TaskAssignee struct {
	TaskID     string    `gorm:"type:uuid;primary_key"`
	UserID     string    `gorm:"type:uuid;primary_key"`
	AssignedBy *string   `gorm:"type:uuid"`
	AssignedAt time.Time `gorm:"default:now()"`
}

// TaskContextSnapshot represents the task_context_snapshots table
type TaskContextSnapshot struct {
	ID          string      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID      string      `gorm:"type:uuid;not null"`
	Role        ContextRole `gorm:"type:context_role;not null"`
	Author      string      `gorm:"type:text"`
	Text        string      `gorm:"type:text"`
	TgMessageID int64       `gorm:"type:bigint"`
	CreatedAt   time.Time   `gorm:"default:now()"`
}

// TaskEvent represents the task_events table
type TaskEvent struct {
	ID        string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TaskID    string    `gorm:"type:uuid;not null"`
	ActorID   *string   `gorm:"type:uuid"`
	Event     string    `gorm:"type:task_event_type;not null"`
	Before    []byte    `gorm:"type:jsonb"`
	After     []byte    `gorm:"type:jsonb"`
	CreatedAt time.Time `gorm:"default:now()"`
}

// TaskComment represents the task_comments table
type TaskComment struct {
	ID        string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskID    string    `gorm:"type:uuid;not null;index" json:"task_id"`
	ParentID  *string   `gorm:"type:uuid;index" json:"parent_id"`
	UserID    string    `gorm:"type:uuid;not null" json:"user_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"default:now()" json:"updated_at"`

	User models.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TaskRepository handles database operations for tasks
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	UpdateStatus(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	ListByUser(ctx context.Context, userID string, filter TaskListFilter) ([]Task, error)
	SoftDelete(ctx context.Context, id string) error

	// Comment methods
	CreateComment(ctx context.Context, comment *TaskComment) (*TaskComment, error)
	ListComments(ctx context.Context, taskID string) ([]TaskComment, error)
	GetByNotionPageID(ctx context.Context, pageID string) (*Task, error)
	ListPendingByGroup(ctx context.Context, groupID string) ([]Task, error)
	ListForReminders(ctx context.Context, now time.Time) ([]Task, error)
	UpdateReminderFlags(ctx context.Context, id string, reminder1h, reminderDue bool) error
}

type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create creates a new task with associations
func (r *taskRepository) Create(ctx context.Context, task *Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID retrieves a task by ID with associations
func (r *taskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
	var task Task
	err := r.db.WithContext(ctx).
		Preload("Assignees").
		Preload("Snapshots").
		First(&task, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}
	return &task, nil
}

// GetByNotionPageID retrieves a task by Notion Page ID
func (r *taskRepository) GetByNotionPageID(ctx context.Context, pageID string) (*Task, error) {
	var task Task
	err := r.db.WithContext(ctx).
		Where("notion_page_id = ?", pageID).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// UpdateStatus updates the status and sync status of a task
func (r *taskRepository) UpdateStatus(ctx context.Context, task *Task) error {
	return r.db.WithContext(ctx).Model(task).Select("Status", "SyncStatus", "NotionPageID", "NotionURL").Updates(task).Error
}

// Update updates main task fields (Title, Description, Status, DueAt, etc)
func (r *taskRepository) Update(ctx context.Context, task *Task) error {
	return r.db.WithContext(ctx).Model(task).Select("Title", "Description", "Status", "SyncStatus", "Topic", "DueAt", "Reminder1hSent", "ReminderDueSent").Updates(task).Error
}

// TaskView represents the type of list view
type TaskView string

const (
	TaskViewAll      TaskView = "all"
	TaskViewAssigned TaskView = "assigned"
	TaskViewCreated  TaskView = "created"
)

// TaskListFilter contains filters for listing tasks
type TaskListFilter struct {
	View       TaskView
	DatabaseID *string
	Limit      int
	Offset     int
}

// ListByUser returns tasks filtered by view/database for the given user
func (r *taskRepository) ListByUser(ctx context.Context, userID string, filter TaskListFilter) ([]Task, error) {
	var tasks []Task

	query := r.db.WithContext(ctx).
		Model(&Task{}).
		Preload("Assignees").
		Preload("Snapshots").
		Where("tasks.deleted_at IS NULL")

	if filter.DatabaseID != nil {
		query = query.Where("tasks.database_id = ?", *filter.DatabaseID)
	}

	switch filter.View {
	case TaskViewAssigned:
		query = query.Group("tasks.id").
			Joins("JOIN task_assignees ta ON ta.task_id = tasks.id").
			Where("ta.user_id = ?", userID)
	case TaskViewCreated:
		query = query.Where("tasks.creator_id = ?", userID)
	default:
		query = query.Group("tasks.id").
			Joins("LEFT JOIN task_assignees ta ON ta.task_id = tasks.id").
			Where("tasks.creator_id = ? OR ta.user_id = ?", userID, userID)
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Order("tasks.created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// SoftDelete performs a soft delete on the task
func (r *taskRepository) SoftDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&Task{}).Error
}

// CreateComment creates a new comment and returns the preloaded version
func (r *taskRepository) CreateComment(ctx context.Context, comment *TaskComment) (*TaskComment, error) {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return nil, err
	}

	var fullComment TaskComment
	err := r.db.WithContext(ctx).
		Preload("User").
		First(&fullComment, "id = ?", comment.ID).Error
	if err != nil {
		return comment, nil // Fallback to non-preloaded
	}
	return &fullComment, nil
}

// ListComments lists comments for a task
func (r *taskRepository) ListComments(ctx context.Context, taskID string) ([]TaskComment, error) {
	var comments []TaskComment
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Preload("User").
		Order("created_at ASC").
		Find(&comments).Error
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// ListPendingByGroup lists tasks in a group that are pending sync
func (r *taskRepository) ListPendingByGroup(ctx context.Context, groupID string) ([]Task, error) {
	var tasks []Task
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND (sync_status = ? OR notion_page_id IS NULL) AND deleted_at IS NULL",
			groupID, TaskSyncStatusPending).
		Preload("Assignees").
		Preload("Snapshots").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListForReminders finds tasks that need reminders sent
func (r *taskRepository) ListForReminders(ctx context.Context, now time.Time) ([]Task, error) {
	var tasks []Task
	// 1. Tasks due in <= 1 hour but reminder_1h not sent
	// 2. Tasks past due but reminder_due not sent
	err := r.db.WithContext(ctx).
		Preload("Assignees").
		Where("status != ? AND due_at IS NOT NULL AND archived = false AND deleted_at IS NULL", TaskStatusDone).
		Where("(due_at <= ? AND reminder_1h_sent = false) OR (due_at <= ? AND reminder_due_sent = false)",
			now.Add(1*time.Hour), now).
		Find(&tasks).Error
	return tasks, err
}

// UpdateReminderFlags updates the reminder sent flags
func (r *taskRepository) UpdateReminderFlags(ctx context.Context, id string, reminder1h, reminderDue bool) error {
	updates := make(map[string]interface{})
	if reminder1h {
		updates["reminder_1h_sent"] = true
	}
	if reminderDue {
		updates["reminder_due_sent"] = true
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(updates).Error
}
