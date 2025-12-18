package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/internal/service/notion"
	"go.uber.org/zap"
)

type mockNotionService struct {
	mock.Mock
}

func (m *mockNotionService) ListDatabases(ctx context.Context, userID string, query string) ([]notion.DatabaseSummary, error) {
	args := m.Called(ctx, userID, query)
	return args.Get(0).([]notion.DatabaseSummary), args.Error(1)
}

func (m *mockNotionService) ValidateDatabase(ctx context.Context, userID string, dbID string) (*notion.ValidationResult, error) {
	args := m.Called(ctx, userID, dbID)
	return args.Get(0).(*notion.ValidationResult), args.Error(1)
}

func (m *mockNotionService) InitializeDatabase(ctx context.Context, userID string, dbID string) (*notion.InitResult, error) {
	return nil, nil
}

func TestListDatabasesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockNotionService)
	logger := zap.NewNop()
	h := NewHandler(logger, service)

	service.On("ListDatabases", mock.Anything, "user-1", "todo").Return([]notion.DatabaseSummary{{ID: "db1", Name: "Tasks"}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/databases?search=todo", nil)
	c.Request = req
	c.Set(middleware.ContextKeyUser, &models.User{ID: "user-1"})

	h.ListDatabases(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	assert.Len(t, items, 1)
}

func TestValidateDatabaseHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := new(mockNotionService)
	logger := zap.NewNop()
	h := NewHandler(logger, service)

	service.On("ValidateDatabase", mock.Anything, "user-1", "db1").Return(&notion.ValidationResult{ID: "db1", Compatible: true}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/databases/db1/validate", nil)
	c.Params = gin.Params{{Key: "database_id", Value: "db1"}}
	c.Request = req
	c.Set(middleware.ContextKeyUser, &models.User{ID: "user-1"})

	h.ValidateDatabase(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
}
