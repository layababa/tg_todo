package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zz/tg_todo/server/internal/task"
)

// TelegramNotifier 通过 Telegram Bot API 发送任务通知。
type TelegramNotifier struct {
	token   string
	apiBase string
	client  *http.Client
}

// NewTelegramNotifier 创建 Telegram 通知器；token 为空时返回 nil 表示禁用。
func NewTelegramNotifier(token, apiBase string) *TelegramNotifier {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		base = "https://api.telegram.org"
	}
	return &TelegramNotifier{
		token:   token,
		apiBase: base,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// TaskStatusChanged 推送任务状态变化消息。
func (n *TelegramNotifier) TaskStatusChanged(ctx context.Context, t *task.Task, actor task.Person, oldStatus, newStatus task.Status) error {
	if n == nil {
		return nil
	}
	if t == nil {
		return fmt.Errorf("nil task")
	}
	if oldStatus == newStatus {
		return nil
	}
	message := buildStatusMessage(t, actor, oldStatus, newStatus)
	return n.broadcast(ctx, t, actor, message)
}

// TaskDeleted 推送任务被删除的提示。
func (n *TelegramNotifier) TaskDeleted(ctx context.Context, t *task.Task, actor task.Person) error {
	if n == nil {
		return nil
	}
	message := fmt.Sprintf("任务《%s》已被 %s 删除。", t.Title, actor.DisplayName)
	return n.broadcast(ctx, t, actor, message)
}

// TaskCreated 推送任务指派通知。
func (n *TelegramNotifier) TaskCreated(ctx context.Context, t *task.Task, creator task.Person) error {
	if n == nil {
		return nil
	}
	if len(t.Assignees) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("任务《%s》已由 %s 分配给你。", t.Title, creator.DisplayName))
	if link := taskLink(t); link != "" {
		sb.WriteString("\n原始消息：")
		sb.WriteString(link)
	}
	var errs []string
	for _, asg := range t.Assignees {
		id, err := strconv.ParseInt(asg.ID, 10, 64)
		if err != nil {
			log.Printf("notify: invalid assignee id %s: %v", asg.ID, err)
			continue
		}
		if err := n.sendMessage(ctx, id, sb.String()); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("task create notify failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (n *TelegramNotifier) broadcast(ctx context.Context, t *task.Task, actor task.Person, message string) error {
	recipients := collectRecipients(t, actor)
	var errs []string
	for _, chatID := range recipients {
		if err := n.sendMessage(ctx, chatID, message); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("telegram notify failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (n *TelegramNotifier) sendMessage(ctx context.Context, chatID int64, text string) error {
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", n.apiBase, n.token)
	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	data.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var body struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.Description != "" {
			return fmt.Errorf("sendMessage %d failed: %s", chatID, body.Description)
		}
		return fmt.Errorf("sendMessage %d failed: %s", chatID, resp.Status)
	}
	return nil
}

func buildStatusMessage(t *task.Task, actor task.Person, oldStatus, newStatus task.Status) string {
	action := "更新"
	switch {
	case oldStatus == task.StatusPending && newStatus == task.StatusCompleted:
		action = "标记完成"
	case oldStatus == task.StatusCompleted && newStatus == task.StatusPending:
		action = "重新打开"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("任务《%s》已由 %s %s。", t.Title, actor.DisplayName, action))
	if link := taskLink(t); link != "" {
		sb.WriteString("\n")
		sb.WriteString("原始消息：")
		sb.WriteString(link)
	}
	return sb.String()
}

func taskLink(t *task.Task) string {
	if t == nil || strings.TrimSpace(t.SourceMessage) == "" {
		return ""
	}
	return t.SourceMessage
}

func collectRecipients(t *task.Task, actor task.Person) []int64 {
	unique := make(map[int64]struct{})
	exclude := actor.ID
	add := func(person task.Person) {
		if strings.TrimSpace(person.ID) == "" {
			return
		}
		if exclude != "" && person.ID == exclude {
			return
		}
		id, err := strconv.ParseInt(person.ID, 10, 64)
		if err != nil {
			log.Printf("notify: invalid user id %s: %v", person.ID, err)
			return
		}
		unique[id] = struct{}{}
	}

	if t != nil {
		add(t.CreatedBy)
		for _, asg := range t.Assignees {
			add(asg)
		}
	}

	recipients := make([]int64, 0, len(unique))
	for id := range unique {
		recipients = append(recipients, id)
	}
	return recipients
}
