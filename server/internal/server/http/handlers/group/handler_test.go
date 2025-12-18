package group

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/models"
	groupservice "github.com/layababa/tg_todo/server/internal/service/group"
	notionsvc "github.com/layababa/tg_todo/server/internal/service/notion"
)

type mockGroupService struct {
	mock.Mock
}

func (m *mockGroupService) ListGroups(ctx context.Context, userID string) ([]groupservice.GroupSummary, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]groupservice.GroupSummary), args.Error(1)
}

func (m *mockGroupService) ValidateDatabase(ctx context.Context, userID, groupID, dbID string) (*notionsvc.ValidationResult, error) {
	args := m.Called(ctx, userID, groupID, dbID)
	return args.Get(0).(*notionsvc.ValidationResult), args.Error(1)
}

func (m *mockGroupService) InitDatabase(ctx context.Context, userID, groupID, dbID string) (*notionsvc.InitResult, error) {
	args := m.Called(ctx, userID, groupID, dbID)
	return args.Get(0).(*notionsvc.InitResult), args.Error(1)
}

func (m *mockGroupService) BindDatabase(ctx context.Context, userID, groupID, dbID string) (*models.Group, error) {
	args := m.Called(ctx, userID, groupID, dbID)
	return args.Get(0).(*models.Group), args.Error(1)
}

func (m *mockGroupService) UnbindDatabase(ctx context.Context, userID, groupID string) (*models.Group, error) {
	args := m.Called(ctx, userID, groupID)
	return args.Get(0).(*models.Group), args.Error(1)
}

func TestListGroupsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockGroupService)
	logger := zap.NewNop()
	h := NewHandler(logger, service, nil) // Task Service optional for List

	service.On("ListGroups", mock.Anything, "user-1").Return([]groupservice.GroupSummary{{ID: "g1"}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Group Handler needs Task Service now
	req := httptest.NewRequest(http.MethodGet, "/groups", nil)
	c.Request = req
	c.Set("userID", "user-1")

	h.ListGroups(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
}

func TestBindGroupForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockGroupService)
	logger := zap.NewNop()
	h := NewHandler(logger, service, nil) // Task Service not needed for this test

	service.On("BindDatabase", mock.Anything, "user-1", "group-1", "db-1").Return((*models.Group)(nil), groupservice.ErrNotAdmin)

	body := strings.NewReader(`{"db_id":"db-1"}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/groups/group-1/bind", body)
	req.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "group_id", Value: "group-1"}}
	c.Request = req
	c.Set("userID", "user-1")

	h.BindGroup(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
