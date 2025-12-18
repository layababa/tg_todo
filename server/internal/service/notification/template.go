package notification

import (
	"fmt"
	"strings"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
)

type EventType string

const (
	EventTaskCreated   EventType = "task_created"
	EventTaskAssigned  EventType = "task_assigned"
	EventStatusChanged EventType = "status_changed"
	EventCommentAdded  EventType = "comment_added"
)

// TemplateData holds data for rendering notification templates
type TemplateData struct {
	Event   EventType
	Task    *repository.Task
	Comment *repository.TaskComment
	Actor   *models.User // Who performed the action
}

// formatMessage formats the notification message based on event type (HTML format)
func formatMessage(data TemplateData) string {
	var sb strings.Builder

	// Actor name
	actorName := ""
	if data.Actor != nil && data.Actor.Name != "" {
		actorName = escapeHTML(data.Actor.Name)
	}

	taskTitle := escapeHTML(data.Task.Title)
	taskLink := fmt.Sprintf(`<a href="https://t.me/todo_app_bot/todo?startapp=task_%s">📂 打开工单</a>`, data.Task.ID)

	switch data.Event {
	case EventTaskCreated:
		sb.WriteString("🆕 <b>新任务</b>\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		if actorName != "" {
			sb.WriteString(fmt.Sprintf("<b>创建者:</b> %s\n", actorName))
		}

	case EventTaskAssigned:
		sb.WriteString("👉 <b>你有新的任务指派</b>\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		if actorName != "" {
			sb.WriteString(fmt.Sprintf("<b>指派人:</b> %s\n", actorName))
		}

	case EventStatusChanged:
		statusText := formatStatusChinese(data.Task.Status)
		sb.WriteString("🔄 <b>任务状态已更新</b>\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		sb.WriteString(fmt.Sprintf("<b>新状态:</b> %s\n", statusText))
		if actorName != "" {
			sb.WriteString(fmt.Sprintf("<b>操作人:</b> %s\n", actorName))
		}

	case EventCommentAdded:
		sb.WriteString("💬 <b>新评论</b>\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		if actorName != "" {
			sb.WriteString(fmt.Sprintf("<b>评论者:</b> %s\n", actorName))
		}
		if data.Comment != nil {
			content := data.Comment.Content
			if len(content) > 100 {
				content = content[:97] + "..."
			}
			sb.WriteString(fmt.Sprintf("\n<i>%s</i>\n", escapeHTML(content)))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s", taskLink))
	return sb.String()
}

// formatStatusChinese converts task status to Chinese
func formatStatusChinese(status repository.TaskStatus) string {
	switch status {
	case repository.TaskStatusToDo:
		return "待办"
	case repository.TaskStatusInProgress:
		return "进行中"
	case repository.TaskStatusDone:
		return "已完成 ✅"
	default:
		return string(status)
	}
}

// escapeHTML escapes special characters for Telegram HTML format
func escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(text)
}
