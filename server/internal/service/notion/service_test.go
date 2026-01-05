package notion

import (
	"context"
	"testing"

	"github.com/dstotijn/go-notion"
	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/security"
)

// MockClient mocks pkgnotion.Client
type MockClient struct {
	mock.Mock
}

func (m *MockClient) CreatePage(ctx context.Context, params pkgnotion.CreatePageParams) (*notion.Page, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notion.Page), args.Error(1)
}

func (m *MockClient) Search(ctx context.Context, query string) (*notion.SearchResponse, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notion.SearchResponse), args.Error(1)
}

func (m *MockClient) GetDatabase(ctx context.Context, id string) (*notion.Database, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notion.Database), args.Error(1)
}

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (m *MockUserRepo) Update(ctx context.Context, user *models.User) error { return nil }
func (m *MockUserRepo) FindByTgID(ctx context.Context, tgID int64) (*models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) FindNotionToken(ctx context.Context, userID string) (*models.UserNotionToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserNotionToken), args.Error(1)
}
func (m *MockUserRepo) SaveNotionToken(ctx context.Context, token *models.UserNotionToken) error {
	return nil
}
func (m *MockUserRepo) ListAll(ctx context.Context) ([]models.User, error) {
	return nil, nil
}

func (m *MockUserRepo) FindByCalendarToken(ctx context.Context, token string) (*models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}

// Ensure MockUserRepo implements repository.UserRepository
var _ repository.UserRepository = (*MockUserRepo)(nil)

func TestListDatabases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockUserRepo := new(MockUserRepo)
	mockClient := new(MockClient)

	encryptionKey := "12345678901234567890123456789012" // 32 chars
	service := NewService(logger, mockUserRepo, encryptionKey)

	// Mock Client Factory
	service.ClientFactory = func(token string) pkgnotion.Client {
		return mockClient
	}

	// Setup User Token
	tokenPlain := "secret_token"
	tokenEnc, _ := security.Encrypt(tokenPlain, encryptionKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{
		UserID:         "user1",
		AccessTokenEnc: tokenEnc,
	}, nil)

	// Setup Search Response
	titleText := "My DB"
	emoji := "📅"

	mockClient.On("Search", mock.Anything, "").Return(&notion.SearchResponse{
		Results: []interface{}{
			&notion.Database{
				ID:    "db1",
				Title: []notion.RichText{{PlainText: titleText}},
				Icon: &notion.Icon{
					Type:  "emoji",
					Emoji: &emoji,
				},
			},
			&notion.Page{
				ID: "page1",
			},
		},
	}, nil)

	// Execute
	dbs, err := service.ListDatabases(context.Background(), "user1", "")

	// Verify
	assert.NoError(t, err)
	assert.Len(t, dbs, 1)
	assert.Equal(t, "db1", dbs[0].ID)
	assert.Equal(t, "My DB", dbs[0].Name)
	assert.Equal(t, "📅", dbs[0].Icon)
}

func TestValidateDatabase_Valid(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockUserRepo := new(MockUserRepo)
	mockClient := new(MockClient)

	encryptionKey := "12345678901234567890123456789012"
	service := NewService(logger, mockUserRepo, encryptionKey)
	service.ClientFactory = func(token string) pkgnotion.Client { return mockClient }

	tokenEnc, _ := security.Encrypt("token", encryptionKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{AccessTokenEnc: tokenEnc}, nil)

	// Mock DB with required fields
	mockClient.On("GetDatabase", mock.Anything, "db1").Return(&notion.Database{
		ID:    "db1",
		Title: []notion.RichText{{PlainText: "Task DB"}},
		Properties: notion.DatabaseProperties{
			"Status":   notion.DatabaseProperty{Type: notion.DBPropTypeStatus},
			"Assignee": notion.DatabaseProperty{Type: notion.DBPropTypePeople},
			"Date":     notion.DatabaseProperty{Type: notion.DBPropTypeDate},
		},
	}, nil)

	res, err := service.ValidateDatabase(context.Background(), "user1", "db1")
	assert.NoError(t, err)
	assert.True(t, res.Compatible)
	assert.Empty(t, res.MissingFields)
}

func TestValidateDatabase_Invalid(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockUserRepo := new(MockUserRepo)
	mockClient := new(MockClient)

	encryptionKey := "12345678901234567890123456789012"
	service := NewService(logger, mockUserRepo, encryptionKey)
	service.ClientFactory = func(token string) pkgnotion.Client { return mockClient }

	tokenEnc, _ := security.Encrypt("token", encryptionKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{AccessTokenEnc: tokenEnc}, nil)

	// Mock DB with missing Date and wrong Status type
	mockClient.On("GetDatabase", mock.Anything, "db1").Return(&notion.Database{
		ID:    "db1",
		Title: []notion.RichText{{PlainText: "Bad DB"}},
		Properties: notion.DatabaseProperties{
			"Status":   notion.DatabaseProperty{Type: notion.DBPropTypeRichText}, // Wrong type
			"Assignee": notion.DatabaseProperty{Type: notion.DBPropTypePeople},
		},
	}, nil)

	res, err := service.ValidateDatabase(context.Background(), "user1", "db1")
	assert.NoError(t, err)
	assert.False(t, res.Compatible)
	assert.Contains(t, res.MissingFields, "Status (Expected Select/Status)")
	assert.Contains(t, res.MissingFields, "Date")
}

func (m *MockClient) UpdateDatabase(ctx context.Context, id string, params notion.UpdateDatabaseParams) (*notion.Database, error) {
	args := m.Called(ctx, id, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notion.Database), args.Error(1)
}

func (m *MockClient) QueryDatabase(ctx context.Context, id string, params *notion.DatabaseQuery) (*notion.DatabaseQueryResponse, error) {
	return nil, nil
}

func (m *MockClient) UpdatePage(ctx context.Context, pageID string, params pkgnotion.UpdatePageParams) (*notion.Page, error) {
	return nil, nil
}

func TestInitializeDatabase_NoMissing(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockUserRepo := new(MockUserRepo)
	mockClient := new(MockClient)
	encryptionKey := "12345678901234567890123456789012"
	service := NewService(logger, mockUserRepo, encryptionKey)
	service.ClientFactory = func(token string) pkgnotion.Client { return mockClient }

	tokenEnc, _ := security.Encrypt("token", encryptionKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{AccessTokenEnc: tokenEnc}, nil)

	mockClient.On("GetDatabase", mock.Anything, "db1").Return(&notion.Database{
		ID: "db1",
		Properties: notion.DatabaseProperties{
			"Status":   notion.DatabaseProperty{Type: notion.DBPropTypeStatus},
			"Assignee": notion.DatabaseProperty{Type: notion.DBPropTypePeople},
			"Date":     notion.DatabaseProperty{Type: notion.DBPropTypeDate},
		},
	}, nil)

	res, err := service.InitializeDatabase(context.Background(), "user1", "db1")
	assert.NoError(t, err)
	assert.False(t, res.Initialized)
	assert.Empty(t, res.CreatedFields)
}

func TestInitializeDatabase_CreatesMissing(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockUserRepo := new(MockUserRepo)
	mockClient := new(MockClient)
	encryptionKey := "12345678901234567890123456789012"
	service := NewService(logger, mockUserRepo, encryptionKey)
	service.ClientFactory = func(token string) pkgnotion.Client { return mockClient }

	tokenEnc, _ := security.Encrypt("token", encryptionKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{AccessTokenEnc: tokenEnc}, nil)

	// Missing Status and Date
	mockClient.On("GetDatabase", mock.Anything, "db1").Return(&notion.Database{
		ID: "db1",
		Properties: notion.DatabaseProperties{
			"Assignee": notion.DatabaseProperty{Type: notion.DBPropTypePeople},
		},
	}, nil)

	// UpdateDatabase expectation
	mockClient.On("UpdateDatabase", mock.Anything, "db1", mock.MatchedBy(func(params notion.UpdateDatabaseParams) bool {
		_, hasStatus := params.Properties["Status"]
		_, hasDate := params.Properties["Date"]
		_, hasAssignee := params.Properties["Assignee"]
		return hasStatus && hasDate && !hasAssignee
	})).Return(&notion.Database{ID: "db1"}, nil)

	res, err := service.InitializeDatabase(context.Background(), "user1", "db1")
	assert.NoError(t, err)
	assert.True(t, res.Initialized)
	assert.Contains(t, res.CreatedFields, "Status")
	assert.Contains(t, res.CreatedFields, "Date")
	assert.NotContains(t, res.CreatedFields, "Assignee")
}
