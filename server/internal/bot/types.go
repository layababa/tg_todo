package bot

// Update represents Telegram update payload.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message represents Telegram message entity.
type Message struct {
	MessageID            int             `json:"message_id"`
	Text                 string          `json:"text"`
	Caption              string          `json:"caption"`
	Chat                 Chat            `json:"chat"`
	From                 *User           `json:"from"`
	ReplyToMessage       *Message        `json:"reply_to_message"`
	ForwardFrom          *User           `json:"forward_from"`
	ForwardFromChat      *Chat           `json:"forward_from_chat"`
	ForwardFromMessageID int             `json:"forward_from_message_id"`
	Entities             []MessageEntity `json:"entities"`
	CaptionEntities      []MessageEntity `json:"caption_entities"`
}

// Chat minimal subset needed to reply/build source links.
type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Title    string `json:"title"`
}

// User minimal subset we need for creator/assignee info.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// MessageEntity captures Telegram entities (mentions, commands, etc.).
type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	User   *User  `json:"user"`
}

// ChatMember is a simplified Telegram chat member payload.
type ChatMember struct {
	Status string `json:"status"`
	User   *User  `json:"user"`
}
