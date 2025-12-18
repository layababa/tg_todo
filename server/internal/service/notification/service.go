package notification

import (
	"context"
	// Added import here
	"github.com/layababa/tg_todo/server/internal/repository"
	"go.uber.org/zap"
)

type TelegramClient interface {
	SendMessage(chatID int64, text string) error
	// Maybe SendMessageMarkdown? For now assume SendMessage handles text.
	// But our template generates Markdown. We might need a generalized Send method or update the client interface.
}

type Service struct {
	logger   *zap.Logger
	repo     repository.TaskRepository
	userRepo repository.UserRepository
	tgClient TelegramClient
}

func NewService(logger *zap.Logger, repo repository.TaskRepository, userRepo repository.UserRepository, tgClient TelegramClient) *Service {
	return &Service{
		logger:   logger,
		repo:     repo,
		userRepo: userRepo,
		tgClient: tgClient,
	}
}

// Notify dispatches a notification for an event
func (s *Service) Notify(ctx context.Context, event EventType, task *repository.Task, actorID string, comment *repository.TaskComment) {
	// 1. Identify recipients
	recipients := make(map[string]bool) // Set of UserIDs

	// Add Creator
	if task.CreatorID != nil && *task.CreatorID != actorID {
		recipients[*task.CreatorID] = true
	}

	// Add Assignees
	for _, assignee := range task.Assignees {
		if assignee.ID != actorID {
			recipients[assignee.ID] = true
		}
	}

	// Don't notify anyone if empty
	if len(recipients) == 0 {
		return
	}

	// Lint said: undefined: repository.User in template.go.
	// This means repository package does NOT expose User.
	// `server/internal/repository/task.go` likely has `User` struct used in `Assignees []User`?
	// Let's assume we need to import `models` in template.go and use `models.User`.

	// Prepare dummy actor for now to fix compile, we will fix imports in next step.
	data := TemplateData{
		Event:   event,
		Task:    task,
		Comment: comment,
		Actor:   nil, // Will fix
	}

	// Fetch Actor (skip if actorID is empty)
	if actorID != "" {
		if act, err := s.userRepo.FindByID(ctx, actorID); err == nil {
			data.Actor = act
		}
	}

	// 3. Render Message
	msg := formatMessage(data)

	// 4. Send to each recipient
	for userID := range recipients {
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			s.logger.Error("failed to find recipient", zap.String("user_id", userID), zap.Error(err))
			continue
		}

		if user.TgID == 0 {
			continue
		}

		if err := s.tgClient.SendMessage(user.TgID, msg); err != nil {
			s.logger.Error("failed to send notification", zap.Int64("chat_id", user.TgID), zap.Error(err))
		}
	}

	s.logger.Info("Notification dispatched",
		zap.String("event", string(event)),
		zap.String("task_id", task.ID),
		zap.Int("recipient_count", len(recipients)))
}
