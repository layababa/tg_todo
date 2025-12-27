package telegram

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	groupsvc "github.com/layababa/tg_todo/server/internal/service/group"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// Mock Repo
type MockUpdateRepo struct{ mock.Mock }

func (m *MockUpdateRepo) Save(ctx context.Context, u *repository.TelegramUpdate) error {
	return m.Called(ctx, u).Error(0)
}

func (m *MockUpdateRepo) GetRecentMessages(ctx context.Context, chatID int64, limit int, beforeID int64) ([]repository.TelegramUpdate, error) {
	args := m.Called(ctx, chatID, limit, beforeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.TelegramUpdate), args.Error(1)
}

// Mock Deduplicator
type MockDeduplicator struct{ mock.Mock }

func (m *MockDeduplicator) IsDuplicate(ctx context.Context, id int64) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// Mock Group Repo
type MockGroupRepo struct{ mock.Mock }

func (m *MockGroupRepo) FindByID(ctx context.Context, id string) (*models.Group, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Group), args.Error(1)
}
func (m *MockGroupRepo) FindByUserID(ctx context.Context, userID string) ([]models.Group, error) {
	return nil, nil // Not used
}
func (m *MockGroupRepo) CreateOrUpdate(ctx context.Context, group *models.Group) error {
	return m.Called(ctx, group).Error(0)
}
func (m *MockGroupRepo) AddMember(ctx context.Context, userID, groupID string, role models.GroupRole) error {
	return m.Called(ctx, userID, groupID, role).Error(0)
}
func (m *MockGroupRepo) IsMember(ctx context.Context, userID, groupID string) (bool, *models.GroupRole, error) {
	return false, nil, nil // Not used in EnsureGroup
}
func (m *MockGroupRepo) ListWithActiveBindings(ctx context.Context) ([]models.Group, error) {
	return nil, nil // Not used
}

func TestHandleWebhook_MyChatMember_Added(t *testing.T) {
	// 1. Mock Telegram Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		// e.g. /token/sendMessage
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	// 2. Setup Deps
	logger := zaptest.NewLogger(t)
	mockRepo := new(MockUpdateRepo)
	mockDedup := new(MockDeduplicator)
	mockGroupRepo := new(MockGroupRepo)

	// Group Service (NotionService nil is fine for EnsureGroup)
	groupService := groupsvc.NewService(logger, mockGroupRepo, nil)

	tgClient := telegram.NewClient("token")
	tgClient.SetBaseURL(ts.URL + "/") // Trailing slash important if client logic expects it? code: ts.baseURL, token. code: "%s%s/%s" -> "URL/token/method"

	mockDedup.On("IsDuplicate", mock.Anything, int64(100)).Return(false, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	// Expect EnsureGroup calls
	mockGroupRepo.On("FindByID", mock.Anything, "200").Return(nil, nil) // Group not found -> Create
	mockGroupRepo.On("CreateOrUpdate", mock.Anything, mock.MatchedBy(func(g *models.Group) bool {
		return g.ID == "200" && g.Title == "Test Group" && g.Status == models.GroupStatusUnbound
	})).Return(nil)
	mockGroupRepo.On("AddMember", mock.Anything, "300", "200", models.GroupRoleAdmin).Return(nil)

	handler := NewHandler(Config{
		Logger:       logger,
		Deduplicator: mockDedup,
		Repo:         mockRepo,
		GroupService: groupService,
		TgClient:     tgClient,
	})

	// 3. Request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{
		"update_id": 100,
		"my_chat_member": {
			"chat": {"id": 200, "title": "Test Group"},
			"from": {"id": 300},
			"new_chat_member": {"status": "administrator"}
		}
	}`
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	c.Request = req

	handler.HandleWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockGroupRepo.AssertExpectations(t)
}

func TestHandleWebhook_Command_Start(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	logger := zaptest.NewLogger(t)
	mockRepo := new(MockUpdateRepo)
	mockDedup := new(MockDeduplicator)

	tgClient := telegram.NewClient("token")
	tgClient.SetBaseURL(ts.URL + "/")

	mockDedup.On("IsDuplicate", mock.Anything, int64(101)).Return(false, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	handler := NewHandler(Config{
		Logger:       logger,
		Deduplicator: mockDedup,
		Repo:         mockRepo,
		TgClient:     tgClient,
		// GroupService nil ok
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{
		"update_id": 101,
		"message": {
			"message_id": 1,
			"from": {"id": 300},
			"chat": {"id": 300, "type": "private"},
			"text": "/start"
		}
	}`
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	c.Request = req

	handler.HandleWebhook(c)
	assert.Equal(t, http.StatusOK, w.Code)
}
