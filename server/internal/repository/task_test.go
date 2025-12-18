package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	createTables := []string{
		`CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			notion_page_id TEXT,
			title TEXT NOT NULL,
			status TEXT,
			sync_status TEXT,
			group_id TEXT,
			database_id TEXT,
			topic TEXT,
			due_at DATETIME,
			creator_id TEXT,
			chat_jump_url TEXT,
			notion_url TEXT,
			archived BOOLEAN DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE TABLE task_assignees (
			task_id TEXT,
			user_id TEXT,
			assigned_by TEXT,
			assigned_at DATETIME
		);`,
		`CREATE TABLE task_context_snapshots (
			id TEXT PRIMARY KEY,
			task_id TEXT,
			role TEXT,
			author TEXT,
			text TEXT,
			tg_message_id INTEGER,
			created_at DATETIME
		);`,
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			tg_id INTEGER,
			deleted_at DATETIME
		);`,
	}
	for _, stmt := range createTables {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func insertTask(t *testing.T, db *gorm.DB, task Task, assigneeIDs ...string) {
	t.Helper()
	require.NoError(t, db.Create(&task).Error)
	for _, userID := range assigneeIDs {
		require.NoError(t, db.Create(&TaskAssignee{
			TaskID: task.ID,
			UserID: userID,
		}).Error)
	}
}

func TestListByUserFiltersViews(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	creatorID := uuid.NewString()
	otherUser := uuid.NewString()
	taskAssigned := Task{ID: uuid.NewString(), Title: "Assigned", CreatorID: &creatorID}
	taskCreated := Task{ID: uuid.NewString(), Title: "Created", CreatorID: &creatorID}

	insertTask(t, db, taskAssigned, otherUser)
	insertTask(t, db, taskCreated)

	ctx := context.Background()

	res, err := repo.ListByUser(ctx, otherUser, TaskListFilter{View: TaskViewAssigned})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, taskAssigned.ID, res[0].ID)

	res, err = repo.ListByUser(ctx, creatorID, TaskListFilter{View: TaskViewCreated})
	require.NoError(t, err)
	require.Len(t, res, 2) // both tasks created by creator
}

func TestSoftDeleteRemovesTask(t *testing.T) {
	db := setupTaskTestDB(t)
	repo := NewTaskRepository(db)

	taskID := uuid.NewString()
	creator := uuid.NewString()
	insertTask(t, db, Task{ID: taskID, Title: "Temp", CreatorID: &creator})

	ctx := context.Background()
	require.NoError(t, repo.SoftDelete(ctx, taskID))

	var count int64
	require.NoError(t, db.Model(&Task{}).Where("id = ?", taskID).Count(&count).Error)
	require.Equal(t, int64(0), count)

	// Ensure soft deleted does not appear
	res, err := repo.ListByUser(ctx, creator, TaskListFilter{View: TaskViewCreated})
	require.NoError(t, err)
	require.Len(t, res, 0)
}
