package notification

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
	"go.uber.org/zap"
)

type TelegramClient interface {
	SendMessage(chatID int64, text string) error
	SendMessageWithButtons(chatID int64, text string, markup telegram.InlineKeyboardMarkup) error
	SendMessageToThread(chatID int64, text string, threadID int) error
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

	// Add Parent Comment Author (for replies)
	if event == EventCommentAdded && comment != nil && comment.ParentID != nil {
		parentComment, err := s.repo.GetCommentByID(ctx, *comment.ParentID)
		if err == nil && parentComment != nil && parentComment.UserID != actorID {
			recipients[parentComment.UserID] = true
		}
	}

	// Don't notify anyone if empty (but continue to Group Sync)
	// if len(recipients) == 0 {
	// 	return
	// }

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

	// 2. Send to Group Chat (Sync Comment)
	if event == EventCommentAdded && task.GroupID != nil && *task.GroupID != "" {
		groupID, err := strconv.ParseInt(*task.GroupID, 10, 64)
		if err == nil {
			threadID := 0
			if task.Topic != "" {
				tid, _ := strconv.ParseInt(task.Topic, 10, 64)
				threadID = int(tid)
			}

			// Build Group Message
			// We need Actor Name. If Actor is passed (even if ID is ""), we use it?
			// The caller of Notify passes actorID. We fetch actor inside Notify for template data.
			// Let's reuse 'data.Actor' if available.
			actorName := "用户"
			if data.Actor != nil {
				actorName = data.Actor.Name
			} else if actorID != "" {
				// Try to fetch if not fetched yet
				if act, err := s.userRepo.FindByID(ctx, actorID); err == nil {
					actorName = act.Name
				}
			}

			// Generate Task Deep Link
			cleanBotName := strings.TrimPrefix(s.botName, "@")
			taskLink := fmt.Sprintf("https://t.me/%s?startapp=task_%s", cleanBotName, task.ID)

			// Format: "💬 新评论 - [Task Title]\n\n[Actor]: [Content]\n[Link]"
			// Using HTML link for title or separate line? User asked for deep link.
			// User example:
			// 💬 新评论 - TaskTitle
			//
			// User: Content
			// (Assume link is attached or implemented via hidden link or button if allowed, but SendMessageToThread no button currently)
			// Let's add a text link at bottom or linked title.
			// User said "Need a deep link to task in group reply".

			groupMsg := fmt.Sprintf("💬 新评论 - <a href=\"%s\">%s</a>\n\n%s: %s",
				taskLink,
				escapeHTML(task.Title),
				actorName,
				comment.Content)

			if err := s.tgClient.SendMessageToThread(groupID, groupMsg, threadID); err != nil {
				s.logger.Error("failed to sync comment to group", zap.Error(err), zap.Int("thread_id", threadID))

				// Retry to General (ThreadID=0) if it was a thread send and failed
				if threadID != 0 {
					s.logger.Info("retrying sync to general topic", zap.Int64("group_id", groupID))
					if err := s.tgClient.SendMessageToThread(groupID, groupMsg, 0); err != nil {
						s.logger.Error("failed to sync comment to general topic fallback", zap.Error(err))
					}
				}
			} else {
				s.logger.Info("synced comment to group", zap.Int64("group_id", groupID), zap.Int("thread_id", threadID))
			}
		}
	}
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
