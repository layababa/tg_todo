package task

// TelegramMessageRef keeps Telegram chat/message identifiers for link-back purposes.
type TelegramMessageRef struct {
	ChatID    int64
	MessageID int64
}

// CreateInput describes the fields required to persist a task.
type CreateInput struct {
	Title            string
	Description      *string
	Creator          Person
	Assignees        []Person
	SourceMessageURL *string
	Status           Status
	TelegramMessage  *TelegramMessageRef
}
