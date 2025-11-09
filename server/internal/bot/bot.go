package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zz/tg_todo/server/internal/task"
)

// Bot 负责与 Telegram API 通讯并调用任务服务。
type Bot struct {
	token       string
	apiBase     string
	client      *http.Client
	taskService *task.Service
	pollTimeout time.Duration
	offset      int
}

// New 实例化 Bot。
func New(token, apiBase string, taskSvc *task.Service) *Bot {
	return &Bot{
		token:       token,
		apiBase:     strings.TrimRight(apiBase, "/"),
		client:      &http.Client{Timeout: 35 * time.Second},
		taskService: taskSvc,
		pollTimeout: 30 * time.Second,
	}
}

// Start 启动长轮询。收到 `/tasks` 指令时会返回当前任务列表。
func (b *Bot) Start(ctx context.Context) error {
	if b.token == "" {
		return fmt.Errorf("bot token is empty")
	}
	log.Println("bot: start polling Telegram updates")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := b.fetchUpdates(ctx)
		if err != nil {
			log.Printf("bot: fetch updates failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, update := range updates {
			b.offset = update.UpdateID + 1
			if err := b.handleUpdate(ctx, update); err != nil {
				log.Printf("bot: handle update failed: %v", err)
			}
		}
	}
}

func (b *Bot) fetchUpdates(ctx context.Context) ([]Update, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/bot%s/getUpdates?timeout=%d&offset=%d", b.apiBase, b.token, int(b.pollTimeout.Seconds()), b.offset),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram response not ok")
	}
	return result.Result, nil
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) error {
	if update.Message == nil {
		return nil
	}
	text := strings.TrimSpace(update.Message.Text)
	switch {
	case strings.HasPrefix(text, "/tasks"):
		return b.replyWithTasks(ctx, update.Message.Chat.ID)
	case strings.HasPrefix(text, "/ping"):
		return b.sendMessage(ctx, update.Message.Chat.ID, "pong ok")
	default:
		return nil
	}
}

func (b *Bot) replyWithTasks(ctx context.Context, chatID int64) error {
	tasks, err := b.taskService.List(ctx)
	if err != nil {
		return b.sendMessage(ctx, chatID, "获取任务失败，请稍后再试。")
	}
	if len(tasks) == 0 {
		return b.sendMessage(ctx, chatID, "暂无相关任务。")
	}
	var buf strings.Builder
	buf.WriteString("当前任务：\n")
	for _, task := range tasks {
		buf.WriteString(fmt.Sprintf("• %s [%s]\n", task.Title, task.Status))
	}
	return b.sendMessage(ctx, chatID, buf.String())
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string) error {
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", b.apiBase, b.token)
	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	data.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendMessage failed: %s", resp.Status)
	}
	return nil
}
