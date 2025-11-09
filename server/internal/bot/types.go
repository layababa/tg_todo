package bot

// Update represents Telegram update payload.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message represents Telegram message entity.
type Message struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      Chat   `json:"chat"`
}

// Chat minimal subset needed to reply.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}
