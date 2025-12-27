package task

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	taskservice "github.com/layababa/tg_todo/server/internal/service/task"
	"go.uber.org/zap"
)

type mockTaskService struct {
	mock.Mock
}

func (m *mockTaskService) ListTasks(ctx context.Context, userID string, params taskservice.ListParams) ([]taskservice.TaskDetail, error) {
	args := m.Called(ctx, userID, params)
	return args.Get(0).([]taskservice.TaskDetail), args.Error(1)
}

func (m *mockTaskService) GetTask(ctx context.Context, id string) (*repository.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Task), args.Error(1)
}

func (m *mockTaskService) DeleteTask(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockTaskService) UpdateTask(ctx context.Context, id string, params taskservice.UpdateParams) (*repository.Task, error) {
	args := m.Called(ctx, id, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Task), args.Error(1)
}

func (m *mockTaskService) CreateWebTask(ctx context.Context, userID, title, description string) (*repository.Task, error) {
	args := m.Called(ctx, userID, title, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Task), args.Error(1)
}

func (m *mockTaskService) CreateComment(ctx context.Context, taskID, userID, content string) (*repository.TaskComment, error) {
	args := m.Called(ctx, taskID, userID, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.TaskComment), args.Error(1)
}

func (m *mockTaskService) ListComments(ctx context.Context, taskID string) ([]repository.TaskComment, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.TaskComment), args.Error(1)
}

func (m *mockTaskService) GetTaskCounts(ctx context.Context, userID string) (*repository.TaskCounts, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.TaskCounts), args.Error(1)
}

type mockUserGroupRepo struct {
	mock.Mock
}

func (m *mockUserGroupRepo) FindByUserAndGroup(ctx context.Context, userID, groupID string) (*models.UserGroup, error) {
	args := m.Called(ctx, userID, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserGroup), args.Error(1)
}

func (m *mockUserGroupRepo) Create(ctx context.Context, userGroup *models.UserGroup) error {
	return m.Called(ctx, userGroup).Error(0)
}

func (m *mockUserGroupRepo) Delete(ctx context.Context, userID, groupID string) error {
	return m.Called(ctx, userID, groupID).Error(0)
}

func TestListTasksHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockTaskService)
	logger := zap.NewNop()
	h := NewHandler(logger, service, new(mockUserGroupRepo))

	params := taskservice.ListParams{View: repository.TaskViewAssigned, Limit: 10}
	service.On("ListTasks", mock.Anything, "user-1", params).Return([]taskservice.TaskDetail{
		{Task: &repository.Task{ID: "task-1"}},
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/tasks?view=assigned&limit=10", nil)
	c.Request = req
	c.Set(middleware.ContextKeyUser, &models.User{ID: "user-1"})

	h.List(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	service.AssertExpectations(t)
}

func TestGetTaskNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockTaskService)
	logger := zap.NewNop()
	h := NewHandler(logger, service, new(mockUserGroupRepo))

	service.On("GetTask", mock.Anything, "task-404").Return((*repository.Task)(nil), gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-404", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "task-404"}}
	c.Request = req
	c.Set(middleware.ContextKeyUser, &models.User{ID: "user-1"})

	h.Get(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteTaskFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockTaskService)
	logger := zap.NewNop()
	h := NewHandler(logger, service, new(mockUserGroupRepo))

	service.On("DeleteTask", mock.Anything, "task-1").Return(errors.New("boom"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "task-1"}}
	c.Request = req
	c.Set(middleware.ContextKeyUser, &models.User{ID: "user-1"})

	h.Delete(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
