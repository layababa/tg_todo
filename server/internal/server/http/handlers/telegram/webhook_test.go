package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldCreateTask(t *testing.T) {
	handler := &Handler{
		botUsername: "test_bot",
	}

	tests := []struct {
		name     string
		msg      *Message
		expected bool
	}{
		{
			name: "群聊中 @Bot 应该创建任务",
			msg: &Message{
				Chat: struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				}{
					ID:   -123456,
					Type: "supergroup",
				},
				Text: "@test_bot @user 修复Bug",
			},
			expected: true,
		},
		{
			name: "群聊中回复消息且不包含当前Bot提及不应该创建任务",
			msg: &Message{
				Chat: struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				}{
					ID:   -123456,
					Type: "group",
				},
				Text: "@user 处理这个",
				ReplyToMessage: &struct {
					MessageID int64 `json:"message_id"`
				}{
					MessageID: 999,
				},
			},
			expected: false,
		},
		{
			name: "群聊中回复消息且包含当前Bot提及应该创建任务",
			msg: &Message{
				Chat: struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				}{
					ID:   -123456,
					Type: "group",
				},
				Text: "@test_bot @user 处理这个",
				ReplyToMessage: &struct {
					MessageID int64 `json:"message_id"`
				}{
					MessageID: 999,
				},
			},
			expected: true,
		},
		{
			name: "私聊应该创建任务",
			msg: &Message{
				Chat: struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				}{
					ID:   123456,
					Type: "private",
				},
				Text: "@test_bot 测试",
			},
			expected: true,
		},
		{
			name: "群聊中普通消息不应该创建任务",
			msg: &Message{
				Chat: struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				}{
					ID:   -123456,
					Type: "supergroup",
				},
				Text: "普通聊天消息",
			},
			expected: false,
		},
		{
			name: "空消息不应该创建任务",
			msg: &Message{
				Chat: struct {
					ID    int64  `json:"id"`
					Type  string `json:"type"`
					Title string `json:"title"`
				}{
					ID:   -123456,
					Type: "supergroup",
				},
				Text: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.shouldCreateTask(tt.msg)
			assert.Equal(t, tt.expected, result)
		})
	}
}
