package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/security"
)

type mockTaskRepository struct {
	mock.Mock
}

func (m *mockTaskRepository) Create(ctx context.Context, task *repository.Task) error {
	return m.Called(ctx, task).Error(0)
}

func (m *mockTaskRepository) GetByID(ctx context.Context, id string) (*repository.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Task), args.Error(1)
}

func (m *mockTaskRepository) UpdateStatus(ctx context.Context, task *repository.Task) error {
	return m.Called(ctx, task).Error(0)
}

func (m *mockTaskRepository) ListByUser(ctx context.Context, userID string, filter repository.TaskListFilter) ([]repository.Task, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.Task), args.Error(1)
}

func (m *mockTaskRepository) SoftDelete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockTaskRepository) Update(ctx context.Context, task *repository.Task) error {
	return m.Called(ctx, task).Error(0)
}

func (m *mockTaskRepository) CreateComment(ctx context.Context, comment *repository.TaskComment) (*repository.TaskComment, error) {
	args := m.Called(ctx, comment)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.TaskComment), args.Error(1)
}

func (m *mockTaskRepository) GetCommentByID(ctx context.Context, id string) (*repository.TaskComment, error) {
	return nil, nil
}

func (m *mockTaskRepository) ListComments(ctx context.Context, taskID string) ([]repository.TaskComment, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.TaskComment), args.Error(1)
}

func (m *mockTaskRepository) GetByNotionPageID(ctx context.Context, pageID string) (*repository.Task, error) {
	args := m.Called(ctx, pageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Task), args.Error(1)
}

func (m *mockTaskRepository) ListPendingByGroup(ctx context.Context, groupID string) ([]repository.Task, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.Task), args.Error(1)
}

func (m *mockTaskRepository) AssignTask(ctx context.Context, taskID, userID string) error {
	return m.Called(ctx, taskID, userID).Error(0)
}

func (m *mockTaskRepository) GetTaskCounts(ctx context.Context, userID string) (*repository.TaskCounts, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.TaskCounts), args.Error(1)
}

func (m *mockTaskRepository) ListForReminders(ctx context.Context, now time.Time) ([]repository.Task, error) {
	args := m.Called(ctx, now)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.Task), args.Error(1)
}

func (m *mockTaskRepository) UpdateReminderFlags(ctx context.Context, id string, reminder1h, reminderDue bool) error {
	return m.Called(ctx, id, reminder1h, reminderDue).Error(0)
}

func TestListTasksDelegatesToRepository(t *testing.T) {
	repo := new(mockTaskRepository)
	service := NewService(ServiceConfig{Repo: repo})

	filter := repository.TaskListFilter{View: repository.TaskViewAssigned, Limit: 20}
	task := repository.Task{ID: "task-1", Title: "Test"}
	repo.On("ListByUser", mock.Anything, "user-1", filter).Return([]repository.Task{task}, nil)

	result, err := service.ListTasks(context.Background(), "user-1", ListParams{
		View:  repository.TaskViewAssigned,
		Limit: 20,
	})
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "task-1", result[0].Task.ID)
	repo.AssertExpectations(t)
}

func TestGetTaskUsesRepository(t *testing.T) {
	repo := new(mockTaskRepository)
	service := NewService(ServiceConfig{Repo: repo})

	task := &repository.Task{ID: "task-1"}
	repo.On("GetByID", mock.Anything, "task-1").Return(task, nil)

	res, err := service.GetTask(context.Background(), "task-1")
	assert.NoError(t, err)
	assert.Equal(t, task, res)
}

func TestDeleteTaskChecksExistence(t *testing.T) {
	repo := new(mockTaskRepository)
	service := NewService(ServiceConfig{Repo: repo})

	repo.On("GetByID", mock.Anything, "task-1").Return(&repository.Task{ID: "task-1"}, nil)
	repo.On("SoftDelete", mock.Anything, "task-1").Return(nil)

	err := service.DeleteTask(context.Background(), "task-1")
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteTaskWhenMissing(t *testing.T) {
	repo := new(mockTaskRepository)
	service := NewService(ServiceConfig{Repo: repo})

	repo.On("GetByID", mock.Anything, "task-404").Return((*repository.Task)(nil), errors.New("not found"))

	err := service.DeleteTask(context.Background(), "task-404")
	assert.Error(t, err)
}

func TestSyncToNotion_Create(t *testing.T) {
	repo := new(mockTaskRepository)
	userRepo := new(mockUserRepo) // Need a real mock here, mockTaskRepository is defined here but mockUserRepo is in creator_test.go
	// Assuming mockUserRepo is visible (same package)

	encryptionKey := "test-key"
	tokenEnc, _ := security.Encrypt("access-token", encryptionKey)

	userRepo.notionTokens = map[string]*models.UserNotionToken{
		"user-1": {UserID: "user-1", AccessTokenEnc: tokenEnc},
	}

	stub := &stubNotionClient{}

	service := NewService(ServiceConfig{
		Repo:          repo,
		UserRepo:      userRepo,
		EncryptionKey: encryptionKey,
		Logger:        zap.NewNop(),
	})
	service.notionClient = func(token string) pkgnotion.Client {
		return stub
	}

	task := &repository.Task{ID: "t1", Title: "New Task", Status: repository.TaskStatusToDo}

	// Expectations
	repo.On("UpdateStatus", mock.Anything, task).Return(nil)

	err := service.SyncToNotion(context.Background(), task, "user-1", "db-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, stub.calls)
	assert.Equal(t, repository.TaskSyncStatusSynced, task.SyncStatus)
	assert.NotNil(t, task.NotionPageID)
}

func TestSyncToNotion_Update(t *testing.T) {
	repo := new(mockTaskRepository)
	userRepo := new(mockUserRepo)

	encryptionKey := "test-key"
	tokenEnc, _ := security.Encrypt("access-token", encryptionKey)

	userRepo.notionTokens = map[string]*models.UserNotionToken{
		"user-1": {UserID: "user-1", AccessTokenEnc: tokenEnc},
	}

	stub := &stubNotionClient{}

	service := NewService(ServiceConfig{
		Repo:          repo,
		UserRepo:      userRepo,
		EncryptionKey: encryptionKey,
		Logger:        zap.NewNop(),
	})
	service.notionClient = func(token string) pkgnotion.Client {
		return stub
	}

	pageID := "page-123"
	task := &repository.Task{
		ID:           "t1",
		Title:        "Updated Task",
		Status:       repository.TaskStatusDone,
		NotionPageID: &pageID,
		SyncStatus:   repository.TaskSyncStatusPending,
	}

	// Expectations
	repo.On("UpdateStatus", mock.Anything, task).Return(nil)

	err := service.SyncToNotion(context.Background(), task, "user-1", "db-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, stub.calls) // Should call UpdatePage (1 call)
	assert.Equal(t, repository.TaskSyncStatusSynced, task.SyncStatus)
}

func TestSyncTaskFromNotion_DeletesArchived(t *testing.T) {
	repo := new(mockTaskRepository)
	service := NewService(ServiceConfig{
		Repo:   repo,
		Logger: zap.NewNop(),
	})

	// Setup: Existing task linked to Notion Page
	existingTask := &repository.Task{ID: "t1", NotionPageID: ptrString("page-123")}
	repo.On("GetByNotionPageID", mock.Anything, "page-123").Return(existingTask, nil)
	repo.On("SoftDelete", mock.Anything, "t1").Return(nil)

	// Execute: Sync with isArchived=true
	err := service.SyncTaskFromNotion(context.Background(), "page-123", "db-1", "Title", "To Do", "url", nil, true)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestSyncTaskFromNotion_IgnoresUnknownArchived(t *testing.T) {
	repo := new(mockTaskRepository)
	service := NewService(ServiceConfig{
		Repo:   repo,
		Logger: zap.NewNop(),
	})

	// Setup: No existing task
	repo.On("GetByNotionPageID", mock.Anything, "page-unknown").Return((*repository.Task)(nil), nil)

	// Execute: Sync with isArchived=true
	err := service.SyncTaskFromNotion(context.Background(), "page-unknown", "db-1", "Title", "To Do", "url", nil, true)
	assert.NoError(t, err)

	// Verify: No SoftDelete called
	repo.AssertNotCalled(t, "SoftDelete")
}

func ptrString(s string) *string {
	return &s
}
