package group

import (
	"context"
	"testing"

	"github.com/dstotijn/go-notion"
	"github.com/layababa/tg_todo/server/internal/models"
	notionsvc "github.com/layababa/tg_todo/server/internal/service/notion"
	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// Mock Group Repo
type MockGroupRepo struct {
	mock.Mock
}

func (m *MockGroupRepo) FindByID(ctx context.Context, id string) (*models.Group, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Group), args.Error(1)
}
func (m *MockGroupRepo) FindByUserID(ctx context.Context, userID string) ([]models.Group, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Group), args.Error(1)
}
func (m *MockGroupRepo) CreateOrUpdate(ctx context.Context, group *models.Group) error {
	return m.Called(ctx, group).Error(0)
}
func (m *MockGroupRepo) AddMember(ctx context.Context, userID, groupID string, role models.GroupRole) error {
	return m.Called(ctx, userID, groupID, role).Error(0)
}
func (m *MockGroupRepo) IsMember(ctx context.Context, userID, groupID string) (bool, *models.GroupRole, error) {
	args := m.Called(ctx, userID, groupID)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	switch v := args.Get(1).(type) {
	case models.GroupRole:
		role := v
		return args.Bool(0), &role, args.Error(2)
	case *models.GroupRole:
		return args.Bool(0), v, args.Error(2)
	default:
		return args.Bool(0), nil, args.Error(2)
	}
}
func (m *MockGroupRepo) ListWithActiveBindings(ctx context.Context) ([]models.Group, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Group), args.Error(1)
}

// Mock User Repo (Minimal for Notion Service)
type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) FindNotionToken(ctx context.Context, userID string) (*models.UserNotionToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserNotionToken), args.Error(1)
}

// Stubs for interface compliance
func (m *MockUserRepo) Create(ctx context.Context, user *models.User) error { return nil }
func (m *MockUserRepo) Update(ctx context.Context, user *models.User) error { return nil }
func (m *MockUserRepo) FindByTgID(ctx context.Context, tgID int64) (*models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) SaveNotionToken(ctx context.Context, token *models.UserNotionToken) error {
	return nil
}
func (m *MockUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) ListAll(ctx context.Context) ([]models.User, error) {
	return nil, nil
}
func (m *MockUserRepo) FindByCalendarToken(ctx context.Context, token string) (*models.User, error) {
	return nil, nil
}

// Mock Notion Client
type MockNotionClient struct {
	mock.Mock
}

func (m *MockNotionClient) CreatePage(ctx context.Context, params pkgnotion.CreatePageParams) (*notion.Page, error) {
	return nil, nil
}
func (m *MockNotionClient) Search(ctx context.Context, query string) (*notion.SearchResponse, error) {
	return nil, nil
}
func (m *MockNotionClient) GetDatabase(ctx context.Context, id string) (*notion.Database, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*notion.Database), args.Error(1)
}
func (m *MockNotionClient) UpdateDatabase(ctx context.Context, id string, params notion.UpdateDatabaseParams) (*notion.Database, error) {
	return nil, nil
}
func (m *MockNotionClient) QueryDatabase(ctx context.Context, id string, params *notion.DatabaseQuery) (*notion.DatabaseQueryResponse, error) {
	return nil, nil
}

func (m *MockNotionClient) UpdatePage(ctx context.Context, pageID string, params pkgnotion.UpdatePageParams) (*notion.Page, error) {
	return nil, nil
}

// Tests
func TestBindDatabase_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockGroupRepo := new(MockGroupRepo)
	mockUserRepo := new(MockUserRepo)
	mockNotionClient := new(MockNotionClient)

	encKey := "12345678901234567890123456789012"
	notionService := notionsvc.NewService(logger, mockUserRepo, encKey)
	notionService.ClientFactory = func(token string) pkgnotion.Client { return mockNotionClient }

	service := NewService(logger, mockGroupRepo, notionService)

	// Setup: Admin check
	mockGroupRepo.On("IsMember", mock.Anything, "user1", "group1").Return(true, models.GroupRoleAdmin, nil)

	// Setup: Notion Token
	tokenEnc, _ := security.Encrypt("token", encKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{AccessTokenEnc: tokenEnc}, nil)

	// Setup: Validate DB (GetDatabase call)
	mockNotionClient.On("GetDatabase", mock.Anything, "db1").Return(&notion.Database{
		ID:    "db1",
		Title: []notion.RichText{{PlainText: "My DB"}},
		Properties: notion.DatabaseProperties{
			"Status":   {Type: notion.DBPropTypeStatus},
			"Assignee": {Type: notion.DBPropTypePeople},
			"Date":     {Type: notion.DBPropTypeDate},
		},
	}, nil)

	// Setup: Find Group
	mockGroupRepo.On("FindByID", mock.Anything, "group1").Return(&models.Group{
		ID:     "group1",
		Status: models.GroupStatusUnbound,
	}, nil)

	// Setup: Update Group
	mockGroupRepo.On("CreateOrUpdate", mock.Anything, mock.MatchedBy(func(g *models.Group) bool {
		return g.ID == "group1" && *g.DatabaseID == "db1" && g.Status == models.GroupStatusConnected
	})).Return(nil)

	// Execute
	group, err := service.BindDatabase(context.Background(), "user1", "group1", "db1")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, "db1", *group.DatabaseID)
}

func TestBindDatabase_NotAdmin(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockGroupRepo := new(MockGroupRepo)
	service := NewService(logger, mockGroupRepo, nil)

	// Setup: Member but not Admin
	mockGroupRepo.On("IsMember", mock.Anything, "user1", "group1").Return(true, models.GroupRoleMember, nil)

	_, err := service.BindDatabase(context.Background(), "user1", "group1", "db1")
	assert.ErrorIs(t, err, ErrNotAdmin)
}

func TestListGroups_ReturnsSummaries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockGroupRepo := new(MockGroupRepo)
	service := NewService(logger, mockGroupRepo, nil)

	group := models.Group{
		ID:           "group1",
		Title:        "Core Team",
		Status:       models.GroupStatusConnected,
		DatabaseID:   ptrString("db1"),
		DatabaseName: "Execution DB",
	}
	mockGroupRepo.On("FindByUserID", mock.Anything, "user1").Return([]models.Group{group}, nil)
	mockGroupRepo.On("IsMember", mock.Anything, "user1", "group1").Return(true, models.GroupRoleAdmin, nil)

	result, err := service.ListGroups(context.Background(), "user1")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "group1", result[0].ID)
	assert.Equal(t, models.GroupRoleAdmin, result[0].Role)
	if assert.NotNil(t, result[0].DB) {
		assert.Equal(t, "db1", result[0].DB.ID)
		assert.Equal(t, "Execution DB", result[0].DB.Name)
	}
}

func TestValidateDatabase_ChecksAdmin(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockGroupRepo := new(MockGroupRepo)
	mockUserRepo := new(MockUserRepo)
	mockNotionClient := new(MockNotionClient)

	encKey := "12345678901234567890123456789012"
	notionService := notionsvc.NewService(logger, mockUserRepo, encKey)
	notionService.ClientFactory = func(string) pkgnotion.Client { return mockNotionClient }
	service := NewService(logger, mockGroupRepo, notionService)

	mockGroupRepo.On("IsMember", mock.Anything, "user1", "group1").Return(true, models.GroupRoleAdmin, nil)
	tokenEnc, _ := security.Encrypt("token", encKey)
	mockUserRepo.On("FindNotionToken", mock.Anything, "user1").Return(&models.UserNotionToken{AccessTokenEnc: tokenEnc}, nil)
	mockNotionClient.On("GetDatabase", mock.Anything, "db1").Return(&notion.Database{
		ID:    "db1",
		Title: []notion.RichText{{PlainText: "DB"}},
		Properties: notion.DatabaseProperties{
			"Status":   notion.DatabaseProperty{Type: notion.DBPropTypeStatus},
			"Assignee": notion.DatabaseProperty{Type: notion.DBPropTypePeople},
			"Date":     notion.DatabaseProperty{Type: notion.DBPropTypeDate},
		},
	}, nil)

	res, err := service.ValidateDatabase(context.Background(), "user1", "group1", "db1")
	assert.NoError(t, err)
	assert.True(t, res.Compatible)
}

func TestInitDatabase_RequiresAdmin(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockGroupRepo := new(MockGroupRepo)
	mockUserRepo := new(MockUserRepo)
	mockNotionClient := new(MockNotionClient)

	encKey := "12345678901234567890123456789012"
	notionService := notionsvc.NewService(logger, mockUserRepo, encKey)
	notionService.ClientFactory = func(string) pkgnotion.Client { return mockNotionClient }
	service := NewService(logger, mockGroupRepo, notionService)

	mockGroupRepo.On("IsMember", mock.Anything, "user1", "group1").Return(false, (*models.GroupRole)(nil), nil)

	_, err := service.InitDatabase(context.Background(), "user1", "group1", "db1")
	assert.ErrorIs(t, err, ErrNotAdmin)
}

func ptrString(s string) *string {
	return &s
}
