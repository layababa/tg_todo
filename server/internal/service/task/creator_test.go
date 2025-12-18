package task

import (
	"context"
	"errors"
	"testing"

	gonotion "github.com/dstotijn/go-notion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
)

func TestCreatorCreateTaskAddsCreatorAndMentions(t *testing.T) {
	t.Parallel()

	mockTaskRepo := &mockTaskRepo{}
	mockUserRepo := &mockUserRepo{
		byTG: map[int64]*models.User{
			111: {
				ID:         "creator-uuid",
				TgID:       111,
				TgUsername: "owner",
			},
		},
		byUsername: map[string]*models.User{
			"alice": {
				ID:         "alice-uuid",
				TgID:       222,
				TgUsername: "alice",
			},
		},
	}

	creator := NewCreator(CreatorConfig{
		Logger:      zap.NewNop(),
		TaskRepo:    mockTaskRepo,
		TaskService: &Service{},
		UpdateRepo:  &mockUpdateRepo{},
		UserRepo:    mockUserRepo,
		GroupRepo:   &mockGroupRepo{},
	})

	taskInput := CreateInput{
		ChatID:    -1001,
		CreatorID: 111,
		Text:      "/todo 修复Webhook @alice",
	}

	taskEntity, err := creator.CreateTask(context.Background(), taskInput)
	require.NoError(t, err)
	require.NotNil(t, taskEntity)
	require.Len(t, mockTaskRepo.createdTasks, 1)

	created := mockTaskRepo.createdTasks[0]
	assert.Equal(t, "修复Webhook", created.Title)
	assert.Equal(t, repository.TaskStatusToDo, created.Status)
	assert.Equal(t, repository.TaskSyncStatusPending, created.SyncStatus)
	require.NotNil(t, created.CreatorID)
	assert.Equal(t, "creator-uuid", *created.CreatorID)
	assert.Len(t, created.Assignees, 2)
	assert.Equal(t, "creator-uuid", created.Assignees[0].ID)
	assert.Equal(t, "alice-uuid", created.Assignees[1].ID)
	assert.Empty(t, created.Snapshots)
}

func TestCreatorCreateTaskIgnoresUnknownMentions(t *testing.T) {
	t.Parallel()

	mockTaskRepo := &mockTaskRepo{}
	mockUserRepo := &mockUserRepo{
		byTG: map[int64]*models.User{
			111: {
				ID:         "creator-uuid",
				TgID:       111,
				TgUsername: "owner",
			},
		},
		usernameErrors: map[string]error{
			"missing": errors.New("not found"),
		},
	}

	creator := NewCreator(CreatorConfig{
		Logger:      zap.NewNop(),
		TaskRepo:    mockTaskRepo,
		TaskService: &Service{},
		UpdateRepo:  &mockUpdateRepo{},
		UserRepo:    mockUserRepo,
		GroupRepo:   &mockGroupRepo{},
	})

	_, err := creator.CreateTask(context.Background(), CreateInput{
		ChatID:    -1001,
		CreatorID: 111,
		Text:      "记录bug @missing",
	})
	require.NoError(t, err)

	require.Len(t, mockTaskRepo.createdTasks, 1)
	created := mockTaskRepo.createdTasks[0]
	assert.Equal(t, "记录bug", created.Title)
	require.Len(t, created.Assignees, 1)
	assert.Equal(t, "creator-uuid", created.Assignees[0].ID)
}

func TestCreatorCreateTaskFailsWhenCreatorLookupFails(t *testing.T) {
	t.Parallel()

	mockTaskRepo := &mockTaskRepo{}
	mockUserRepo := &mockUserRepo{
		findErr: errors.New("db down"),
	}

	creator := NewCreator(CreatorConfig{
		Logger:      zap.NewNop(),
		TaskRepo:    mockTaskRepo,
		TaskService: &Service{},
		UpdateRepo:  &mockUpdateRepo{},
		UserRepo:    mockUserRepo,
		GroupRepo:   &mockGroupRepo{},
	})

	_, err := creator.CreateTask(context.Background(), CreateInput{
		ChatID:    1,
		CreatorID: 999,
		Text:      "/todo nothing",
	})
	require.Error(t, err)
	assert.Empty(t, mockTaskRepo.createdTasks)
}

func TestParseCommandRemovesTodoPrefixAndMentions(t *testing.T) {
	t.Parallel()

	c := &Creator{}
	title, mentions := c.parseCommand("/todo Deploy Preview @alice @bob")

	assert.Equal(t, "Deploy Preview", title)
	assert.Equal(t, []string{"@alice", "@bob"}, mentions)
}

type mockTaskRepo struct {
	createdTasks      []*repository.Task
	err               error
	updateErr         error
	updateStatusCalls int
	lastUpdatedTask   *repository.Task
}

func (m *mockTaskRepo) Create(_ context.Context, task *repository.Task) error {
	if m.err != nil {
		return m.err
	}
	m.createdTasks = append(m.createdTasks, task)
	return nil
}

func (m *mockTaskRepo) GetByID(_ context.Context, id string) (*repository.Task, error) {
	for _, task := range m.createdTasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, errors.New("not implemented")
}

func (m *mockTaskRepo) UpdateStatus(_ context.Context, task *repository.Task) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updateStatusCalls++
	m.lastUpdatedTask = task
	return nil
}

func (m *mockTaskRepo) Update(_ context.Context, task *repository.Task) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.lastUpdatedTask = task
	return nil
}

func (m *mockTaskRepo) CreateComment(_ context.Context, _ *repository.TaskComment) error {
	return nil
}

func (m *mockTaskRepo) ListComments(_ context.Context, _ string) ([]repository.TaskComment, error) {
	return nil, nil
}

func (m *mockTaskRepo) ListByUser(context.Context, string, repository.TaskListFilter) ([]repository.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) GetByNotionPageID(context.Context, string) (*repository.Task, error) {
	return nil, nil
}

func (m *mockTaskRepo) SoftDelete(context.Context, string) error {
	return nil
}

func (m *mockTaskRepo) ListPendingByGroup(context.Context, string) ([]repository.Task, error) {
	return nil, nil
}

type mockUserRepo struct {
	byTG           map[int64]*models.User
	byUsername     map[string]*models.User
	findErr        error
	usernameErrors map[string]error
	notionTokens   map[string]*models.UserNotionToken
	notionTokenErr error
}

func (m *mockUserRepo) FindByTgID(_ context.Context, tgID int64) (*models.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if user, ok := m.byTG[tgID]; ok {
		return user, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	return &models.User{ID: id, Name: "Mock User"}, nil
}

func (m *mockUserRepo) Create(context.Context, *models.User) error {
	return nil
}

func (m *mockUserRepo) Update(context.Context, *models.User) error {
	return nil
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (*models.User, error) {
	if m.usernameErrors != nil {
		if err, ok := m.usernameErrors[username]; ok {
			return nil, err
		}
	}
	if user, ok := m.byUsername[username]; ok {
		return user, nil
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) FindNotionToken(_ context.Context, userID string) (*models.UserNotionToken, error) {
	if m.notionTokenErr != nil {
		return nil, m.notionTokenErr
	}
	if m.notionTokens != nil {
		if token, ok := m.notionTokens[userID]; ok {
			return token, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) SaveNotionToken(context.Context, *models.UserNotionToken) error {
	return nil
}

func (m *mockUserRepo) ListAll(context.Context) ([]models.User, error) {
	return nil, nil
}

type mockUpdateRepo struct {
	err error
}

func (m *mockUpdateRepo) Save(context.Context, *repository.TelegramUpdate) error {
	return m.err
}

func (m *mockUpdateRepo) GetRecentMessages(context.Context, int64, int) ([]repository.TelegramUpdate, error) {
	return nil, nil // Return empty list for tests by default
}

type stubNotionClient struct {
	calls      int
	lastParams pkgnotion.CreatePageParams
	createErr  error
}

func (s *stubNotionClient) CreatePage(ctx context.Context, params pkgnotion.CreatePageParams) (*gonotion.Page, error) {
	s.calls++
	s.lastParams = params
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &gonotion.Page{
		ID:  "page-123",
		URL: "https://notion.so/page-123",
	}, nil
}

func (s *stubNotionClient) Search(context.Context, string) (*gonotion.SearchResponse, error) {
	return nil, nil
}

func (s *stubNotionClient) GetDatabase(context.Context, string) (*gonotion.Database, error) {
	return nil, nil
}

func (s *stubNotionClient) UpdateDatabase(context.Context, string, gonotion.UpdateDatabaseParams) (*gonotion.Database, error) {
	return nil, nil
}

func (s *stubNotionClient) QueryDatabase(context.Context, string, *gonotion.DatabaseQuery) (*gonotion.DatabaseQueryResponse, error) {
	return nil, nil
}

func (s *stubNotionClient) UpdatePage(ctx context.Context, pageID string, params pkgnotion.UpdatePageParams) (*gonotion.Page, error) {
	s.calls++
	return &gonotion.Page{ID: pageID}, nil
}

// ... Stub methods not used in remaining tests but struct might be used if I add back tests
// Leave it or remove? Removed usage in deleted tests.
// But Service uses pkgnotion.Client interface.
// Creator test does not use it anymore.

// ... group repo mocks/stubs

type mockGroupRepo struct {
	group *models.Group
	err   error
}

func (m *mockGroupRepo) FindByID(ctx context.Context, id string) (*models.Group, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.group, nil
}

func (m *mockGroupRepo) FindByUserID(_ context.Context, _ string) ([]models.Group, error) {
	return nil, nil
}
func (m *mockGroupRepo) CreateOrUpdate(_ context.Context, _ *models.Group) error { return nil }
func (m *mockGroupRepo) AddMember(_ context.Context, _, _ string, _ models.GroupRole) error {
	return nil
}
func (m *mockGroupRepo) IsMember(_ context.Context, _, _ string) (bool, *models.GroupRole, error) {
	return false, nil, nil
}

func (m *mockGroupRepo) ListWithActiveBindings(_ context.Context) ([]models.Group, error) {
	return nil, nil // Not used
}
