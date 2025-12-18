package scheduler

import (
	"context"
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
)

type Service struct {
	logger   *zap.Logger
	cron     *cron.Cron
	userRepo repository.UserRepository
	taskRepo repository.TaskRepository
	tgClient *telegram.Client
}

func NewService(logger *zap.Logger, userRepo repository.UserRepository, taskRepo repository.TaskRepository, tgClient *telegram.Client) *Service {
	return &Service{
		logger:   logger,
		cron:     cron.New(),
		userRepo: userRepo,
		taskRepo: taskRepo,
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

	s.cron.Start()
}

func (s *Service) Stop() {
	s.cron.Stop()
}

// SendDailyDigest manually triggers the digest (can be called by Cron or Debug)
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
	// List pending tasks (To Do + In Progress)
	// We iterate views to get all. Or we can just use TaskViewAll and filter in memory?
	// But ListByUser filters by Creator OR Assignee.
	// TaskViewAll + Limit 50 should be enough for digest.

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
		// Verify if we should send "No tasks" message?
		// PRD usually implies silence is golden, or maybe a "You're all clear!"
		// Let's settle for silence to avoid spam, unless User configured "Send even if empty".
		// For now: Silence.
		return
	}

	// Build Message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 每日摘要 (Daily Digest)\n\n您有 %d 个待办任务：\n", len(pendingTasks)))

	// Show top 10
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

	// Send
	if err := s.tgClient.SendMessage(chatID, sb.String()); err != nil {
		s.logger.Error("failed to send digest", zap.Int64("chat_id", chatID), zap.Error(err))
	}
}
