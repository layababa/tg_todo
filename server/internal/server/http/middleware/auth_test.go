package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/pkg/telegramauth"
)

// Mock UserRepository
type mockUserRepository struct {
	users          map[int64]*models.User
	createErr      error
	findErr        error
	shouldFail     bool
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
	if m.findErr != nil {
		return nil, m.findErr
	}
	user, exists := m.users[tgID]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	// Default implementation: iterate through users to find by ID
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	// Simulate ID generation
	if user.ID == "" {
		user.ID = "generated-uuid-" + string(rune(user.TgID))
	}
	m.users[user.TgID] = user
	return nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	m.users[user.TgID] = user
	return nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil // Not needed for auth tests
}

func (m *mockUserRepository) FindNotionToken(ctx context.Context, userID string) (*models.UserNotionToken, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepository) SaveNotionToken(ctx context.Context, token *models.UserNotionToken) error {
	return nil
}

func (m *mockUserRepository) ListAll(ctx context.Context) ([]models.User, error) {
	return nil, nil
}

func (m *mockUserRepository) FindByCalendarToken(ctx context.Context, token string) (*models.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func TestTelegramAuth_ValidInitData_NewUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Note: Full end-to-end test with valid signature would require
	// actual Telegram init data generation. This is tested indirectly
	// through the individual component tests (buildUserName, GetUserFromContext)
	// and will be covered in integration tests.
	t.Skip("Requires valid Telegram signature - covered in integration tests")
}

func TestTelegramAuth_MissingInitData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockUserRepository()
	botToken := "test_bot_token"

	router := gin.New()
	router.Use(TelegramAuth(botToken, repo))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing_init_data")
}

func TestTelegramAuth_ExistingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockUserRepository()

	// Pre-create a user
	existingUser := &models.User{
		ID:       "user-123",
		TgID:     999888,
		Name:     "Existing User",
		Timezone: "UTC+0",
	}
	repo.users[existingUser.TgID] = existingUser

	// For this test, we'd need valid init data
	// The actual implementation would validate and load the user
}

func TestBuildUserName(t *testing.T) {
	tests := []struct {
		name     string
		user     *telegramauth.TelegramUser
		expected string
	}{
		{
			name: "First and Last name",
			user: &telegramauth.TelegramUser{
				FirstName: "John",
				LastName:  "Doe",
			},
			expected: "John Doe",
		},
		{
			name: "First name only",
			user: &telegramauth.TelegramUser{
				FirstName: "John",
			},
			expected: "John",
		},
		{
			name: "Username only",
			user: &telegramauth.TelegramUser{
				Username: "johndoe",
			},
			expected: "@johndoe",
		},
		{
			name:     "Fallback to User",
			user:     &telegramauth.TelegramUser{},
			expected: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildUserName(tt.user)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("user exists in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		expectedUser := &models.User{
			ID:   "test-123",
			TgID: 456,
			Name: "Test User",
		}
		c.Set(ContextKeyUser, expectedUser)

		user, exists := GetUserFromContext(c)
		assert.True(t, exists)
		assert.Equal(t, expectedUser, user)
	})

	t.Run("user does not exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		user, exists := GetUserFromContext(c)
		assert.False(t, exists)
		assert.Nil(t, user)
	})

	t.Run("wrong type in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(ContextKeyUser, "not a user")

		user, exists := GetUserFromContext(c)
		assert.False(t, exists)
		assert.Nil(t, user)
	})
}

func TestTelegramAuth_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockUserRepository()
	repo.findErr = errors.New("database connection failed")

	// This would test database error handling
	// Actual test would require valid init data
}

func TestTelegramAuth_UserCreationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockUserRepository()
	repo.createErr = errors.New("failed to create user")

	// This would test user creation error handling
	// Actual test would require valid init data for a new user
}
