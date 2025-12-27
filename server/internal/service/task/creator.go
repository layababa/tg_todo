package task

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
)

// Creator handles task creation logic
type Creator struct {
	logger      *zap.Logger
	taskRepo    repository.TaskRepository
	taskService *Service // Use Service for Sync
	updateRepo  repository.TelegramUpdateRepository
	userRepo    repository.UserRepository
	groupRepo   repository.GroupRepository
	pendingRepo repository.PendingAssignmentRepository
}

// CreatorConfig holds configuration for Creator
type CreatorConfig struct {
	Logger      *zap.Logger
	TaskRepo    repository.TaskRepository
	TaskService *Service
	UpdateRepo  repository.TelegramUpdateRepository
	UserRepo    repository.UserRepository
	GroupRepo   repository.GroupRepository
	PendingRepo repository.PendingAssignmentRepository
}

// NewCreator creates a new task creator
func NewCreator(cfg CreatorConfig) *Creator {
	return &Creator{
		logger:      cfg.Logger,
		taskRepo:    cfg.TaskRepo,
		taskService: cfg.TaskService,
		updateRepo:  cfg.UpdateRepo,
		userRepo:    cfg.UserRepo,
		groupRepo:   cfg.GroupRepo,
		pendingRepo: cfg.PendingRepo,
	}
}

// CreateInput represents input for creating a task
type CreateInput struct {
	ChatID    int64
	CreatorID int64 // Telegram User ID
	Text      string
	ChatTitle string // Optional: For group creation if missing
	ChatType  string
	ReplyToID int64 // Optional: Message ID being replied to
}

// CreateTask creates a task from telegram input
func (c *Creator) CreateTask(ctx context.Context, input CreateInput) (*repository.Task, []string, error) {
	// 1. Find or Create Creator User
	creator, err := c.userRepo.FindByTgID(ctx, input.CreatorID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get creator: %w", err)
	}

	// 2. Parse Text (Title & Assignees)
	title, assigneeNames := c.parseCommand(input.Text)

	// 3. Resolve Assignees
	var assignees []models.User
	var pendingAssignees []string

	for _, name := range assigneeNames {
		// Remove @ prefix
		username := strings.TrimPrefix(name, "@")
		user, err := c.userRepo.GetByUsername(ctx, username)
		if err != nil {
			c.logger.Warn("failed to find assignee", zap.String("username", username), zap.Error(err))
			pendingAssignees = append(pendingAssignees, name)
			continue
		}
		if user != nil {
			assignees = append(assignees, *user)
		}
	}

	// 78. Fallback: If no assignees (and no pending), assign to creator
	if len(assignees) == 0 && len(pendingAssignees) == 0 {
		assignees = append(assignees, *creator)
	}

	// 4. Capture Context (Last 10 messages)
	snapshots, err := c.captureContext(ctx, input.ChatID, input.ReplyToID)
	if err != nil {
		c.logger.Warn("failed to capture context", zap.Error(err))
		// Continue without context
	}

	// 5. Resolve Group & Database
	var groupID *string
	var databaseID *string
	groupIDStr := fmt.Sprintf("%d", input.ChatID)

	group, err := c.groupRepo.FindByID(ctx, groupIDStr)
	// If err is 'not found', we proceed as Unbound (nil groupID).
	// Assuming FindByID returns nil, nil or specific error for not found.
	// Standard repo usually returns ErrNotFound.
	// For now, let's log warn and proceed.
	if err == nil && group != nil {
		groupID = &group.ID
		databaseID = group.DatabaseID

		// Update Group Title if changed
		if input.ChatTitle != "" && group.Title != input.ChatTitle {
			group.Title = input.ChatTitle
			if err := c.groupRepo.CreateOrUpdate(ctx, group); err != nil {
				c.logger.Warn("failed to update group title", zap.Error(err))
			} else {
				c.logger.Info("updated group title", zap.String("id", *groupID), zap.String("new_title", input.ChatTitle))
			}
		}
	} else {
		// Group not found? If we have ChatTitle and it looks like a group, let's create it as Unbound.
		// This ensures we can display the source name.
		if input.ChatTitle != "" && (input.ChatType == "group" || input.ChatType == "supergroup") {
			newGroup := &models.Group{
				ID:        groupIDStr,
				Title:     input.ChatTitle,
				Status:    models.GroupStatusUnbound,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := c.groupRepo.CreateOrUpdate(ctx, newGroup); err == nil {
				groupID = &newGroup.ID
				c.logger.Info("auto-created unbound group for task", zap.String("id", groupIDStr), zap.String("title", input.ChatTitle))
			} else {
				c.logger.Warn("failed to auto-create group", zap.Error(err))
			}
		} else {
			c.logger.Debug("group not found and not created", zap.String("chat_id", groupIDStr))
		}
	}

	// 6. Create Task in DB (DB FIRST)
	task := &repository.Task{
		Title:      title,
		Status:     repository.TaskStatusToDo,
		SyncStatus: repository.TaskSyncStatusPending,
		CreatorID:  &creator.ID,
		Assignees:  assignees,
		Snapshots:  snapshots,
		GroupID:    groupID,
		DatabaseID: databaseID,
	}

	// Generate Jump URL
	if input.ReplyToID != 0 {
		task.ChatJumpURL = c.generateJumpURL(input.ChatID, input.ReplyToID)
	} else if len(snapshots) > 0 {
		// Use the last snapshot's message ID as jump target
		lastMsgID := snapshots[len(snapshots)-1].TgMessageID
		task.ChatJumpURL = c.generateJumpURL(input.ChatID, lastMsgID)
	}

	if err := c.taskRepo.Create(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 7. Sync to Notion
	if creator.NotionConnected && databaseID != nil && *databaseID != "" {
		// Use Service to Sync
		go func() {
			if err := c.taskService.SyncToNotion(context.Background(), task, creator.ID, *databaseID); err != nil {
				c.logger.Error("async sync failed", zap.Error(err))
			}
		}()
	} else {
		c.logger.Info("skipping notion sync",
			zap.Bool("user_connected", creator.NotionConnected),
			zap.Any("database_id", databaseID))
	}

	// 7. Store Pending Assignments
	if len(pendingAssignees) > 0 && c.pendingRepo != nil {
		for _, name := range pendingAssignees {
			username := strings.TrimPrefix(name, "@")
			pa := &models.PendingAssignment{
				TaskID:     task.ID,
				TgUsername: username,
			}
			if err := c.pendingRepo.Create(ctx, pa); err != nil {
				c.logger.Error("failed to create pending assignment", zap.String("username", username), zap.Error(err))
				// We don't fail the task creation, just log error
			}
		}
	}

	return task, pendingAssignees, nil
}

func (c *Creator) generateJumpURL(chatID int64, messageID int64) string {
	// Telegram Deep Link Format: https://t.me/c/CHAT_ID/MESSAGE_ID
	// For supergroups (starting with -100), we need to extract the ID part.

	chatIDStr := fmt.Sprintf("%d", chatID)
	if strings.HasPrefix(chatIDStr, "-100") {
		chatIDStr = strings.TrimPrefix(chatIDStr, "-100")
	} else if strings.HasPrefix(chatIDStr, "-") {
		// Normal groups, usually deep linking via c/ works if user is member?
		// Actually t.me/c/ is for private supergroups/channels.
		// For private basic groups, deep linking is harder without invite link.
		// But let's assume Supergroups for now as they are most common for persistent chats.
		chatIDStr = strings.TrimPrefix(chatIDStr, "-")
	}

	return fmt.Sprintf("https://t.me/c/%s/%d", chatIDStr, messageID)
}

// parseCommand extracts title and mentions from text
// Example: "@Bot fix bug @alice" -> Title: "fix bug", Mentions: ["@alice"]
func (c *Creator) parseCommand(text string) (string, []string) {
	// Regex to find mentions
	re := regexp.MustCompile(`@\w+`)
	mentions := re.FindAllString(text, -1)

	// Remove mentions from text to get title
	title := re.ReplaceAllString(text, "")
	title = strings.TrimSpace(title)

	// Remove leading command if present (e.g. /todo)
	title = strings.TrimPrefix(title, "/todo")
	title = strings.TrimSpace(title)

	return title, mentions
}

// captureContext retrieves recent messages from telegram_updates
func (c *Creator) captureContext(ctx context.Context, chatID int64, replyToID int64) ([]repository.TaskContextSnapshot, error) {
	updates, err := c.updateRepo.GetRecentMessages(ctx, chatID, 10)
	if err != nil {
		return nil, err
	}

	var snapshots []repository.TaskContextSnapshot
	// Reverse to chronological order (Oldest -> Newest)
	for i := len(updates) - 1; i >= 0; i-- {
		u := updates[i]

		var payload struct {
			Message struct {
				MessageID int64 `json:"message_id"`
				From      struct {
					ID        int64  `json:"id"`
					FirstName string `json:"first_name"`
				} `json:"from"`
				Text string `json:"text"`
			} `json:"message"`
		}

		if err := json.Unmarshal(u.RawData, &payload); err != nil {
			continue // Skip broken updates
		}

		if payload.Message.Text == "" {
			continue
		}

		role := repository.ContextRoleOther
		// We don't have easy way to know "ME" unless we pass actorID to captureContext.
		// For now default to 'other'.
		// If we want "Me", we check if payload.Message.From.ID == creatorID (which is not passed here).
		// We can update signature later if needed. For now "Other" is fine or display Name.

		snapshots = append(snapshots, repository.TaskContextSnapshot{
			Role:        role,
			Author:      payload.Message.From.FirstName,
			Text:        payload.Message.Text,
			TgMessageID: payload.Message.MessageID,
		})
	}

	return snapshots, nil
}

// CreatePersonalTask creates a new personal task from a forwarded message
func (c *Creator) CreatePersonalTask(ctx context.Context, input CreateInput, meta map[string]interface{}) (*repository.Task, error) {
	// 1. Find or Create Creator User
	creator, err := c.userRepo.FindByTgID(ctx, input.CreatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator: %w", err)
	}

	// 2. Check for Default Personal Database (Story U1)
	var databaseID *string
	if creator.DefaultDatabaseID != nil && *creator.DefaultDatabaseID != "" {
		databaseID = creator.DefaultDatabaseID
	}

	// 3. Create Task
	task := &repository.Task{
		Title:      input.Text,
		Status:     repository.TaskStatusToDo,
		SyncStatus: repository.TaskSyncStatusPending,
		CreatorID:  &creator.ID,
		DatabaseID: databaseID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if source, ok := meta["source"].(string); ok {
		task.Description = "Forwarded source: " + source
	}

	if err := c.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 4. Sync to Notion (Optional)
	if creator.NotionConnected && databaseID != nil && *databaseID != "" {
		go func() {
			if err := c.taskService.SyncToNotion(context.Background(), task, creator.ID, *databaseID); err != nil {
				c.logger.Error("async sync failed", zap.Error(err))
			}
		}()
	}

	return task, nil
}
