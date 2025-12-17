package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/pkg/notion"
)

const testEncryptionKey = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE="

func TestGetStatusReturnsUserProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepository()
	handler := newTestHandler(t, repo, &stubOAuthService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/status", nil)
	user := &models.User{
		ID:              "user-123",
		TgID:            98765,
		Name:            "Test User",
		PhotoURL:        "https://t.me/i/userpic.jpg",
		Timezone:        "UTC+8",
		NotionConnected: true,
	}
	c.Set(middleware.ContextKeyUser, user)

	handler.GetStatus(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]any)
	require.Equal(t, true, data["notion_connected"])
	userData := data["user"].(map[string]any)
	require.Equal(t, float64(user.TgID), userData["tg_id"])
	require.Equal(t, "Test User", userData["name"])
}

func TestGetNotionAuthURLReturnsSignedState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepository()
	oauth := &stubOAuthService{
		config: notion.OAuthConfig{
			ClientID:    "client_1",
			RedirectURI: "https://miniapp.local/callback",
		},
	}
	handler := newTestHandler(t, repo, oauth)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/notion/url", nil)
	user := &models.User{ID: "user-456", TgID: 12345, Name: "State User"}
	c.Set(middleware.ContextKeyUser, user)

	handler.GetNotionAuthURL(c)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	rawURL := data["url"].(string)
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	state := parsed.Query().Get("state")
	require.NotEmpty(t, state)

	tgID, err := handler.stateCodec.Decode(state)
	require.NoError(t, err)
	require.Equal(t, user.TgID, tgID)
	require.Equal(t, state, oauth.lastState)
}

func TestNotionCallbackPersistsToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepository()
	user := &models.User{
		ID:       "user-789",
		TgID:     5555,
		Name:     "OAuth User",
		Timezone: "UTC",
	}
	repo.users[user.TgID] = user

	oauth := &stubOAuthService{
		exchangeResp: &notion.TokenResponse{
			AccessToken:   "secret-token",
			WorkspaceID:   "ws_x",
			WorkspaceName: "Workspace X",
		},
	}

	handler := newTestHandler(t, repo, oauth)
	state, err := handler.stateCodec.Encode(user.TgID)
	require.NoError(t, err)

	body := fmt.Sprintf(`{"code":"abc123","state":"%s"}`, state)
	req := httptest.NewRequest(http.MethodPost, "/auth/notion/callback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.NotionCallback(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, repo.savedToken)
	require.Equal(t, user.ID, repo.savedToken.UserID)
	require.True(t, repo.users[user.TgID].NotionConnected)
}

func TestNotionCallbackInvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockUserRepository()
	handler := newTestHandler(t, repo, &stubOAuthService{})

	body := `{"code":"abc123","state":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/notion/callback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.NotionCallback(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHMACStateCodecRoundTrip(t *testing.T) {
	codec, err := newHMACStateCodec(testEncryptionKey)
	require.NoError(t, err)

	state, err := codec.Encode(987654321)
	require.NoError(t, err)

	decoded, err := codec.Decode(state)
	require.NoError(t, err)
	require.Equal(t, int64(987654321), decoded)
}

func TestHMACStateCodecRejectsTamperedState(t *testing.T) {
	codec, err := newHMACStateCodec(testEncryptionKey)
	require.NoError(t, err)

	state, err := codec.Encode(42)
	require.NoError(t, err)

	raw, err := base64.RawURLEncoding.DecodeString(state)
	require.NoError(t, err)
	raw[0] = '9'
	tampered := base64.RawURLEncoding.EncodeToString(raw)

	_, err = codec.Decode(tampered)
	require.Error(t, err)
}

// --- test helpers ---

type mockUserRepository struct {
	users          map[int64]*models.User
	savedToken     *models.UserNotionToken
	saveTokenErr   error
	updateErr      error
	findByTgIDFunc func(ctx context.Context, tgID int64) (*models.User, error)
	findByIDFunc   func(ctx context.Context, id string) (*models.User, error)
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[int64]*models.User),
	}
}

func (m *mockUserRepository) FindByTgID(ctx context.Context, tgID int64) (*models.User, error) {
	if m.findByTgIDFunc != nil {
		return m.findByTgIDFunc(ctx, tgID)
	}
	user, ok := m.users[tgID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	m.users[user.TgID] = user
	return nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.users[user.TgID] = user
	return nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil // Not needed for auth tests
}

func (m *mockUserRepository) FindNotionToken(ctx context.Context, userID string) (*models.UserNotionToken, error) {
	if m.savedToken != nil && m.savedToken.UserID == userID {
		return m.savedToken, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepository) SaveNotionToken(ctx context.Context, token *models.UserNotionToken) error {
	if m.saveTokenErr != nil {
		return m.saveTokenErr
	}
	m.savedToken = token
	return nil
}

func (m *mockUserRepository) ListAll(ctx context.Context) ([]models.User, error) {
	return nil, nil
}

type stubOAuthService struct {
	config       notion.OAuthConfig
	lastState    string
	exchangeResp *notion.TokenResponse
	exchangeErr  error
	receivedCode string
}

func (s *stubOAuthService) GenerateAuthURL(state string) string {
	s.lastState = state
	return notion.GenerateAuthURL(s.config, state)
}

func (s *stubOAuthService) ExchangeCode(ctx context.Context, code string) (*notion.TokenResponse, error) {
	s.receivedCode = code
	if s.exchangeErr != nil {
		return nil, s.exchangeErr
	}
	return s.exchangeResp, nil
}

func newTestHandler(t *testing.T, repo repository.UserRepository, oauth oauthService) *Handler {
	t.Helper()
	handler, err := NewHandler(Config{
		UserRepo:      repo,
		OAuthService:  oauth,
		EncryptionKey: testEncryptionKey,
	})
	require.NoError(t, err)
	return handler
}
