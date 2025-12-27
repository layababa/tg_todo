package notification

import (
	"fmt"
	"strings"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
)

type EventType string

const (
	EventTaskCreated         EventType = "task_created"
	EventTaskAssigned        EventType = "task_assigned"
	EventStatusChanged       EventType = "status_changed"
	EventCommentAdded        EventType = "comment_added"
	EventTaskAssigneeChanged EventType = "assignee_changed" // New Event
	EventReminder1h          EventType = "reminder_1h"
	EventReminderDue         EventType = "reminder_due"
)

type RecipientRole string

const (
	RoleCreator  RecipientRole = "creator"
	RoleAssignee RecipientRole = "assignee"
)

// TemplateData holds data for rendering notification templates
type TemplateData struct {
	Event         EventType
	Task          *repository.Task
	Comment       *repository.TaskComment
	Actor         *models.User  // Who performed the action
	RecipientRole RecipientRole // Role of the person receiving the notification
	BotName       string        // Telegram Bot Username
	AppShortName  string        // Mini App Short Name (from BotFather)
	ContextInfo   string        // Generic info (e.g. "From X to Y")
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

	case EventTaskAssigneeChanged:
		sb.WriteString("👤 <b>负责人已变更</b>\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		if data.ContextInfo != "" {
			sb.WriteString(fmt.Sprintf("<b>变更:</b> %s\n", data.ContextInfo))
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

	case EventReminder1h:
		sb.WriteString("⏰ <b>任务即将到期</b> (1小时后)\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		if data.RecipientRole == RoleCreator {
			sb.WriteString("\n💡 请记得及时验收该任务。")
		} else {
			sb.WriteString("\n💡 请记得及时完成并提交。")
		}

	case EventReminderDue:
		sb.WriteString("🚨 <b>任务已到达截止时间</b>\n\n")
		sb.WriteString(fmt.Sprintf("<b>任务:</b> %s\n", taskTitle))
		if data.RecipientRole == RoleCreator {
			sb.WriteString("\n💡 该任务已到期，请检查进度或进行验收。")
		} else {
			sb.WriteString("\n💡 该任务已到期，请尽快完成并更新状态。")
		}
	}

	return sb.String()
}

// BuildTaskMarkup creates the inline keyboard for a task
func BuildTaskMarkup(taskID, botName, appShortName string) telegram.InlineKeyboardMarkup {
	if botName == "" {
		return telegram.InlineKeyboardMarkup{}
	}

	// Ensure botName doesn't have @ for the URL
	cleanBotName := strings.TrimPrefix(botName, "@")

	// Use the MOST compatible format: https://t.me/botname?startapp=xxx
	// This format is universally supported and always loads the bot's Main App.
	// It bypasses potential URL routing issues with secondary "Direct Links".
	url := fmt.Sprintf("https://t.me/%s?startapp=task_%s", cleanBotName, taskID)

	return telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: "📂 打开工单",
					URL:  url,
				},
			},
		},
	}
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
