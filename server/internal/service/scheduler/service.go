package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/service/notification"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
)

type Service struct {
	logger   *zap.Logger
	cron     *cron.Cron
	userRepo repository.UserRepository
	taskRepo repository.TaskRepository
	notifier *notification.Service
	tgClient *telegram.Client
}

func NewService(logger *zap.Logger, userRepo repository.UserRepository, taskRepo repository.TaskRepository, notifier *notification.Service, tgClient *telegram.Client) *Service {
	return &Service{
		logger:   logger,
		cron:     cron.New(),
		userRepo: userRepo,
		taskRepo: taskRepo,
		notifier: notifier,
		tgClient: tgClient,
	}
}

func (s *Service) Start() {
	// Schedule Daily Digest at 9:00 AM
	_, err := s.cron.AddFunc("0 9 * * *", func() {
		s.logger.Info("running daily digest job")
		s.SendDailyDigest(context.Background())
	})
	if err != nil {
		s.logger.Error("failed to schedule daily digest", zap.Error(err))
	} else {
		s.logger.Info("daily digest scheduled for 09:00 AM")
	}

	// Schedule Task Reminders every minute
	_, err = s.cron.AddFunc("* * * * *", func() {
		s.CheckReminders(context.Background())
	})
	if err != nil {
		s.logger.Error("failed to schedule task reminders", zap.Error(err))
	}

	s.cron.Start()
}

func (s *Service) Stop() {
	s.cron.Stop()
}

func (s *Service) SendDailyDigest(ctx context.Context) {
	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		s.logger.Error("failed to list users for digest", zap.Error(err))
		return
	}

	for _, user := range users {
		if user.TgID == 0 {
			continue
		}
		s.processUserDigest(ctx, user.ID, user.TgID)
	}
}

func (s *Service) processUserDigest(ctx context.Context, userID string, chatID int64) {
	filter := repository.TaskListFilter{
		View:  repository.TaskViewAll,
		Limit: 100,
	}
	tasks, err := s.taskRepo.ListByUser(ctx, userID, filter)
	if err != nil {
		s.logger.Error("failed to list tasks for user", zap.String("user_id", userID), zap.Error(err))
		return
	}

	var pendingTasks []repository.Task
	for _, t := range tasks {
		if t.Status != repository.TaskStatusDone {
			pendingTasks = append(pendingTasks, t)
		}
	}

	if len(pendingTasks) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 每日摘要 (Daily Digest)\n\n您有 %d 个待办任务：\n", len(pendingTasks)))

	limit := 10
	for i, t := range pendingTasks {
		if i >= limit {
			sb.WriteString(fmt.Sprintf("\n...还有 %d 个任务", len(pendingTasks)-limit))
			break
		}

		icon := "⬜"
		if t.Status == repository.TaskStatusInProgress {
			icon = "🔄"
		}

		sb.WriteString(fmt.Sprintf("\n%d. %s %s", i+1, icon, t.Title))
	}

	sb.WriteString("\n\n💪 加油！输入 /todo 添加新任务。")

	if err := s.tgClient.SendMessage(chatID, sb.String()); err != nil {
		s.logger.Error("failed to send digest", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}

// CheckReminders scans for tasks that need reminders
func (s *Service) CheckReminders(ctx context.Context) {
	now := time.Now()
	tasks, err := s.taskRepo.ListForReminders(ctx, now)
	if err != nil {
		s.logger.Error("failed to list tasks for reminders", zap.Error(err))
		return
	}

	for _, task := range tasks {
		s.processTaskReminders(ctx, task, now)
	}
}

func (s *Service) processTaskReminders(ctx context.Context, task repository.Task, now time.Time) {
	remind1h := false
	remindDue := false

	if task.DueAt == nil {
		return
	}

	// 1. Check 1h reminder
	oneHourFromNow := now.Add(1 * time.Hour)
	if task.DueAt.Before(oneHourFromNow) && !task.Reminder1hSent {
		remind1h = true
	}

	// 2. Check due reminder
	if task.DueAt.Before(now) && !task.ReminderDueSent {
		remindDue = true
	}

	if !remind1h && !remindDue {
		return
	}

	// Send notifications
	if s.notifier != nil {
		if remindDue {
			s.notifier.NotifyReminder(ctx, notification.EventReminderDue, &task)
		} else if remind1h {
			s.notifier.NotifyReminder(ctx, notification.EventReminder1h, &task)
		}
	}

	// Update flags
	if err := s.taskRepo.UpdateReminderFlags(ctx, task.ID, remind1h, remindDue); err != nil {
		s.logger.Error("failed to update reminder flags", zap.String("task_id", task.ID), zap.Error(err))
	}
}

