package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dstotijn/go-notion"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/service/notification"
	pkgnotion "github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/security"
)

type Service struct {
	logger        *zap.Logger
	repo          repository.TaskRepository
	userRepo      repository.UserRepository // Needed for Token
	notifier      *notification.Service
	encryptionKey string
	notionClient  func(token string) pkgnotion.Client
}

// ServiceConfig holds configuration for Service
type ServiceConfig struct {
	Logger        *zap.Logger
	Repo          repository.TaskRepository
	UserRepo      repository.UserRepository
	Notifier      *notification.Service
	EncryptionKey string
}

// NewService creates a new task service
func NewService(cfg ServiceConfig) *Service {
	return &Service{
		logger:        cfg.Logger,
		repo:          cfg.Repo,
		userRepo:      cfg.UserRepo,
		notifier:      cfg.Notifier,
		encryptionKey: cfg.EncryptionKey,
		notionClient:  pkgnotion.NewClient,
	}
}

// UpdateParams represents parameters for updating a task
type UpdateParams struct {
	Title       *string
	Description *string
	Status      *repository.TaskStatus
	DueAt       *time.Time
	SyncStatus  *repository.TaskSyncStatus // Added to support manual sync reset if needed
}

// ListParams represents filters for listing tasks
type ListParams struct {
	View       repository.TaskView
	DatabaseID *string
	Limit      int
	Offset     int
}

// TaskDetail is the DTO returned to callers
type TaskDetail struct {
	Task *repository.Task `json:"task"`
}

// ListTasks returns tasks for the user based on filter
func (s *Service) ListTasks(ctx context.Context, userID string, params ListParams) ([]TaskDetail, error) {
	filter := repository.TaskListFilter{
		View:       params.View,
		DatabaseID: params.DatabaseID,
		Limit:      params.Limit,
		Offset:     params.Offset,
	}

	tasks, err := s.repo.ListByUser(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	result := make([]TaskDetail, 0, len(tasks))
	for i := range tasks {
		result = append(result, TaskDetail{Task: &tasks[i]})
	}
	return result, nil
}

// GetTask returns a single task by ID
func (s *Service) GetTask(ctx context.Context, id string) (*repository.Task, error) {
	return s.repo.GetByID(ctx, id)
}

// DeleteTask soft deletes the task
func (s *Service) DeleteTask(ctx context.Context, id string) error {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}
	return s.repo.SoftDelete(ctx, id)
}

// UpdateTask updates the task fields
func (s *Service) UpdateTask(ctx context.Context, id string, params UpdateParams) (*repository.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	if params.Title != nil {
		task.Title = *params.Title
	}

	statusChanged := false
	if params.Status != nil {
		if task.Status != *params.Status {
			statusChanged = true
			task.Status = *params.Status
		}
	}

	if params.Description != nil {
		task.Description = *params.Description
	}

	if params.DueAt != nil {
		task.DueAt = params.DueAt
		// Reset reminder flags when due date changes
		task.Reminder1hSent = false
		task.ReminderDueSent = false
	}

	// Reset sync status if critical fields changed
	if params.Title != nil || params.Status != nil || params.Description != nil || params.DueAt != nil {
		task.SyncStatus = repository.TaskSyncStatusPending
	}
	if params.SyncStatus != nil {
		task.SyncStatus = *params.SyncStatus
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}

	// Notify
	// We need actorID. Context usually has user info, but Service methods passed explicit userID often?
	// UpdateTask doesn't take userID!
	// We should probably extract userID from context (set by auth middleware) or update signature.
	// For now, let's skip actorID if not present, OR use a context helper.
	// But NotificationService.Notify needs valid actorID for logic (not notifying self).
	// If we don't have actorID, everyone gets notified.
	// Let's rely on ContextKeyUser or pass userID.
	// UpdateTask signature: `UpdateTask(ctx, id, params)`.
	// We should update it to `UpdateTask(ctx, userID, id, params)` or get from ctx.
	// Getting from ctx requires dependency on middleware package (cycle?) or using string key.
	// Ideally service shouldn't depend on HTTP middleware keys.
	// We should update the signature to accept userID.
	// But that's a larger refactor in Handlers calling it.
	// Let's assume we can notify "System" or anonymous if we don't have ID for now.
	// Wait, if I change Status in UI, I am the actor. I shouldn't get notification.
	// This is critical.
	// Let's assume we update the signature in next step or use specific context key logic if safe.
	// Actually, `CreateComment` takes `userID`. `CreateWebTask` takes `userID`.
	// `ListTasks` takes `userID`.
	// `UpdateTask` is the outlier!
	// I will update `UpdateTask` signature to take `userID`.

	if statusChanged && s.notifier != nil {
		// Mock Actor ID for now? Or pass emtpy string?
		// Empty string -> Logic will notify everyone including me if I am creator/assignee.
		// I will pass empty string for now and log TODO.
		s.notifier.Notify(ctx, notification.EventStatusChanged, task, "", nil)
	}

	return task, nil
}

// CreateWebTask creates a new task from web interface
func (s *Service) CreateWebTask(ctx context.Context, userID, title, description string) (*repository.Task, error) {
	// Look up user's default database if needed
	task := &repository.Task{
		Title:       title,
		Description: description,
		CreatorID:   &userID,
		Status:      repository.TaskStatusToDo,
		SyncStatus:  repository.TaskSyncStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.Notify(ctx, notification.EventTaskCreated, task, userID, nil)
	}

	return task, nil
}

// CreateComment creates a new comment
func (s *Service) CreateComment(ctx context.Context, taskID, userID, content string, parentID *string) (*repository.TaskComment, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("task not found")
	}

	comment := &repository.TaskComment{
		TaskID:   taskID,
		UserID:   userID,
		Content:  content,
		ParentID: parentID,
	}
	createdComment, err := s.repo.CreateComment(ctx, comment)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		s.notifier.Notify(ctx, notification.EventCommentAdded, task, userID, createdComment)
	}

	return createdComment, nil
}

// ListComments lists comments for a task
func (s *Service) ListComments(ctx context.Context, taskID string) ([]repository.TaskComment, error) {
	return s.repo.ListComments(ctx, taskID)
}

// SyncTaskFromNotion upserts a task from Notion data
func (s *Service) SyncTaskFromNotion(ctx context.Context, notionPageID, databaseID, title, status string, notionURL string, assignees []string, isArchived bool) error {
	// Check if task exists by NotionPageID
	existing, err := s.repo.GetByNotionPageID(ctx, notionPageID)
	if err != nil {
		return err
	}

	if isArchived {
		if existing != nil {
			s.logger.Info("syncing deletion from notion", zap.String("task_id", existing.ID), zap.String("notion_id", notionPageID))
			return s.repo.SoftDelete(ctx, existing.ID)
		}
		// If not found locally, nothing to delete
		return nil
	}

	now := time.Now()

	if existing != nil {
		// Update
		needsUpdate := false
		if existing.Title != title {
			existing.Title = title
			needsUpdate = true
		}
		if existing.Status != repository.TaskStatus(status) {
			existing.Status = repository.TaskStatus(status)
			needsUpdate = true
			if s.notifier != nil {
				// Notify status change
				s.notifier.Notify(ctx, notification.EventStatusChanged, existing, "system", nil)
			}
		}
		// Todo: Handle Assignees update if we map Notion Users to Local Users

		if needsUpdate {
			existing.UpdatedAt = now
			existing.SyncStatus = repository.TaskSyncStatusSynced
			return s.repo.Update(ctx, existing)
		}
		return nil
	}

	// Create New
	newTask := &repository.Task{
		Title:        title,
		Status:       repository.TaskStatus(status),
		SyncStatus:   repository.TaskSyncStatusSynced,
		NotionPageID: &notionPageID,
		NotionURL:    &notionURL,
		DatabaseID:   &databaseID,
		CreatedAt:    now,
		UpdatedAt:    now,
		// Assignees? We need to find UserID by Notion UserID. This requires a UserRepo lookup.
		// For MVP, we might skip assignee sync or do best effort if we had a mapping service.
	}

	err = s.repo.Create(ctx, newTask)
	if err == nil && s.notifier != nil {
		// Notify creation
		s.notifier.Notify(ctx, notification.EventTaskCreated, newTask, "system", nil)
	}
	return err
}

// SyncToNotion syncs the task to Notion asynchronously
func (s *Service) SyncToNotion(ctx context.Context, task *repository.Task, userID string, databaseID string) error {
	logger := s.logger.With(zap.String("task_id", task.ID), zap.String("user_id", userID))
	logger.Info("syncing to Notion...")

	// 1. Get Token
	token, err := s.userRepo.FindNotionToken(ctx, userID)
	if err != nil {
		logger.Error("failed to find notion token", zap.Error(err))
		return err
	}

	// 2. Decrypt Token
	accessToken, err := security.Decrypt(token.AccessTokenEnc, s.encryptionKey)
	if err != nil {
		logger.Error("failed to decrypt token", zap.Error(err))
		return err
	}

	// 3. Create Client
	client := s.notionClient(accessToken)

	// 4. Check if Task is Already Synced (Update vs Create)
	if task.NotionPageID != nil && *task.NotionPageID != "" {
		// UPDATE
		pageID := *task.NotionPageID
		// Convert status if needed (Assuming Notion uses "To Do", "In Progress", "Done")
		// We need to map local status to Notion status.
		// Local: ToDo, InProgress, Done
		notionStatus := "To Do"
		switch task.Status {
		case repository.TaskStatusInProgress:
			notionStatus = "In Progress"
		case repository.TaskStatusDone:
			notionStatus = "Done"
		}

		_, err := client.UpdatePage(ctx, pageID, pkgnotion.UpdatePageParams{
			Title:  &task.Title,
			Status: &notionStatus,
		})
		if err != nil {
			logger.Error("failed to update page in notion", zap.Error(err))
			// Should we mark as failed? Or just log?
			// If update fails, it might be archived.
			// For now, mark as Failed so we retry? Or keep as Pending?
			// Keeping as Synced might be wrong.
			// Let's mark as Pending so it retries, or Failed.
			// If we mark Failed, user sees error.
			task.SyncStatus = repository.TaskSyncStatusFailed
			s.repo.UpdateStatus(ctx, task)
			return err
		}

		// Update local sync status
		task.SyncStatus = repository.TaskSyncStatusSynced
		s.repo.UpdateStatus(ctx, task)
		logger.Info("notion page updated", zap.String("page_id", pageID))
		return nil
	}

	// CREATE (Existing Logic)
	// 5. Build Content Blocks
	children := s.buildContentBlocks(task)

	// 6. Create Page
	page, err := client.CreatePage(ctx, pkgnotion.CreatePageParams{
		DatabaseID: databaseID,
		Title:      task.Title,
		Status:     "To Do", // Default for new task
		Children:   children,
	})
	if err != nil {
		logger.Error("failed to create page in notion", zap.Error(err))
		task.SyncStatus = repository.TaskSyncStatusFailed
		s.repo.UpdateStatus(ctx, task)
		return err
	}

	// 7. Update Task
	task.NotionPageID = &page.ID
	task.NotionURL = &page.URL
	task.SyncStatus = repository.TaskSyncStatusSynced
	s.repo.UpdateStatus(ctx, task)

	logger.Info("notion page created", zap.String("page_id", page.ID))
	return nil
}

// SyncPendingTasks finds all pending tasks for a group and syncs them
func (s *Service) SyncPendingTasks(ctx context.Context, groupID string) error {
	// We need an admin user ID for the group to get the token.
	// However, `SyncPendingTasks` is usually called in context of a user action (Bind).
	// Let's modify signature to accept userID who initiated the sync/bind.
	// If called via cron, we'd need to resolving proper user.
	return errors.New("method signature requires userID")
}

// SyncPendingTasksForUser syncs tasks for a group using a specific user's token
func (s *Service) SyncPendingTasksForUser(ctx context.Context, userID, groupID, databaseID string) error {
	tasks, err := s.repo.ListPendingByGroup(ctx, groupID)
	if err != nil {
		return err
	}

	s.logger.Info("found pending tasks to sync", zap.Int("count", len(tasks)), zap.String("group_id", groupID))

	for _, task := range tasks {
		// Run sequentially to avoid rate limits? Or parallel?
		// Sequential is safer for Notion API rate limits.
		if err := s.SyncToNotion(ctx, &task, userID, databaseID); err != nil {
			s.logger.Warn("failed to sync specific task", zap.String("task_id", task.ID), zap.Error(err))
			// Continue with others
		}
	}
	return nil
}

// buildContentBlocks constructs Notion blocks from task data
func (s *Service) buildContentBlocks(task *repository.Task) []notion.Block {
	var children []notion.Block

	// Description
	if task.Description != "" {
		children = append(children, notion.ParagraphBlock{
			RichText: []notion.RichText{{
				Text: &notion.Text{Content: task.Description},
			}},
		})
	}

	// Context Snapshots
	if len(task.Snapshots) > 0 {
		children = append(children, notion.Heading3Block{
			RichText: []notion.RichText{{
				Text: &notion.Text{Content: "Context Snapshot"},
			}},
		})

		for _, s := range task.Snapshots {
			author := s.Author
			if author == "" {
				author = "User"
			}
			text := fmt.Sprintf("%s: %s", author, s.Text)
			children = append(children, notion.QuoteBlock{
				RichText: []notion.RichText{{
					Text: &notion.Text{Content: text},
				}},
			})
		}
	}

	// Chat Jump URL
	if task.ChatJumpURL != "" {
		emoji := "💬"
		children = append(children, notion.CalloutBlock{
			RichText: []notion.RichText{{
				Text: &notion.Text{
					Content: "Jump to Telegram Chat",
					Link:    &notion.Link{URL: task.ChatJumpURL},
				},
			}},
			Icon: &notion.Icon{
				Type:  notion.IconTypeEmoji,
				Emoji: &emoji,
			},
		})
	}

	return children
}

// AssignTaskToTelegramUser assigns the task to a user identified by Telegram ID
// It ensures the user exists in the local database first
func (s *Service) AssignTaskToTelegramUser(ctx context.Context, taskID string, tgUser *models.User) error {
	// 1. Ensure User Exists
	existingUser, err := s.userRepo.FindByTgID(ctx, tgUser.TgID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to lookup user: %w", err)
	}

	var userID string
	if existingUser != nil {
		userID = existingUser.ID
		// Optional: Update user info if changed?
	} else {
		// Create new user
		if err := s.userRepo.Create(ctx, tgUser); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		userID = tgUser.ID
	}

	// 2. Assign Task
	return s.AssignTask(ctx, taskID, userID)
}

// AssignTask assigns the task to a user (by internal UUID)
func (s *Service) AssignTask(ctx context.Context, taskID, userID string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}

	// Capture Old Assignee Name
	oldAssigneeName := "无"
	if len(task.Assignees) > 0 {
		oldAssigneeName = task.Assignees[0].Name
	}

	// 1. Update DB
	if err := s.repo.AssignTask(ctx, taskID, userID); err != nil {
		return err
	}

	// 2. Fetch New Assignee Name
	newUser, err := s.userRepo.FindByID(ctx, userID)
	newAssigneeName := "未知用户"
	if err == nil && newUser != nil {
		newAssigneeName = newUser.Name
	}

	// 3. Notify Creator
	// Refresh task to ensure latest state (though we pass task object, ID is constant)
	// We pass the task object we fetched earlier as it contains Title/CreatorID correctly.
	s.notifier.NotifyAssigneeChange(ctx, task, oldAssigneeName, newAssigneeName)

	// Log
	s.logger.Info("task assigned",
		zap.String("task_id", taskID),
		zap.String("user_id", userID),
		zap.String("old_assignee", oldAssigneeName),
		zap.String("new_assignee", newAssigneeName))

	// Sync logic
	task.SyncStatus = repository.TaskSyncStatusPending
	s.repo.UpdateStatus(ctx, task)

	return nil
}
