package notification

import (
	"context"
	"fmt"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
	"go.uber.org/zap"
)

type TelegramClient interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithButtons(chatID int64, text string, markup telegram.InlineKeyboardMarkup) error
}

type Service struct {
	logger       *zap.Logger
	repo         repository.TaskRepository
	userRepo     repository.UserRepository
	tgClient     TelegramClient
	botName      string
	appShortName string
}

func NewService(logger *zap.Logger, repo repository.TaskRepository, userRepo repository.UserRepository, tgClient TelegramClient, botName, appShortName string) *Service {
	return &Service{
		logger:       logger,
		repo:         repo,
		userRepo:     userRepo,
		tgClient:     tgClient,
		botName:      botName,
		appShortName: appShortName,
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

	// Prepare template data
	data := TemplateData{
		Event:        event,
		Task:         task,
		Comment:      comment,
		Actor:        nil,
		BotName:      s.botName,
		AppShortName: s.appShortName,
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

		markup := BuildTaskMarkup(task.ID, s.botName, s.appShortName)
		s.logger.Info("sending notification",
			zap.Int64("chat_id", user.TgID),
			zap.String("url", markup.InlineKeyboard[0][0].URL))

		if markup.InlineKeyboard != nil {
			if err := s.tgClient.SendMessageWithButtons(user.TgID, msg, markup); err != nil {
				s.logger.Error("failed to send notification with buttons", zap.Int64("chat_id", user.TgID), zap.Error(err))
			}
		} else {
			if err := s.tgClient.SendMessage(user.TgID, msg); err != nil {
				s.logger.Error("failed to send notification", zap.Int64("chat_id", user.TgID), zap.Error(err))
			}
		}
	}

	s.logger.Info("Notification dispatched",
		zap.String("event", string(event)),
		zap.String("task_id", task.ID),
		zap.Int("recipient_count", len(recipients)))
}

// NotifyReminder sends differentiated reminders to creator and assignees
func (s *Service) NotifyReminder(ctx context.Context, event EventType, task *repository.Task) {
	markup := BuildTaskMarkup(task.ID, s.botName, s.appShortName)
	s.logger.Info("preparing reminders", zap.String("url", markup.InlineKeyboard[0][0].URL))

	// 1. Creator
	if task.CreatorID != nil {
		user, err := s.userRepo.FindByID(ctx, *task.CreatorID)
		if err == nil && user.TgID != 0 {
			msg := formatMessage(TemplateData{
				Event:         event,
				Task:          task,
				RecipientRole: RoleCreator,
				BotName:       s.botName,
				AppShortName:  s.appShortName,
			})
			if markup.InlineKeyboard != nil {
				_ = s.tgClient.SendMessageWithButtons(user.TgID, msg, markup)
			} else {
				_ = s.tgClient.SendMessage(user.TgID, msg)
			}
		}
	}

	// 2. Assignees
	for _, assignee := range task.Assignees {
		// Avoid double notification if creator is also an assignee
		if task.CreatorID != nil && assignee.ID == *task.CreatorID {
			continue
		}

		user, err := s.userRepo.FindByID(ctx, assignee.ID)
		if err == nil && user.TgID != 0 {
			msg := formatMessage(TemplateData{
				Event:         event,
				Task:          task,
				RecipientRole: RoleAssignee,
				BotName:       s.botName,
				AppShortName:  s.appShortName,
			})
			if markup.InlineKeyboard != nil {
				_ = s.tgClient.SendMessageWithButtons(user.TgID, msg, markup)
			} else {
				_ = s.tgClient.SendMessage(user.TgID, msg)
			}
		}
	}

	s.logger.Info("Reminders dispatched",
		zap.String("event", string(event)),
		zap.String("task_id", task.ID))
}

// NotifyAssigneeChange notifies the creator of the assignee change
func (s *Service) NotifyAssigneeChange(ctx context.Context, task *repository.Task, oldAssignee, newAssignee string) {
	// Only notify Creator if they exist
	if task.CreatorID == nil {
		return
	}

	creator, err := s.userRepo.FindByID(ctx, *task.CreatorID)
	if err != nil || creator == nil || creator.TgID == 0 {
		return
	}

	// Format "From X to Y"
	contextInfo := fmt.Sprintf("由 %s 更改为 %s", oldAssignee, newAssignee)
	if oldAssignee == "" {
		contextInfo = fmt.Sprintf("指派给 %s", newAssignee)
	}

	msg := formatMessage(TemplateData{
		Event:        EventTaskAssigneeChanged,
		Task:         task,
		ContextInfo:  contextInfo,
		BotName:      s.botName,
		AppShortName: s.appShortName,
	})

	markup := BuildTaskMarkup(task.ID, s.botName, s.appShortName)

	if markup.InlineKeyboard != nil {
		_ = s.tgClient.SendMessageWithButtons(creator.TgID, msg, markup)
	} else {
		_ = s.tgClient.SendMessage(creator.TgID, msg)
	}
}
