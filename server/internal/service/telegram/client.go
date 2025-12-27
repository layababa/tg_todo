package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const telegramAPIBase = "https://api.telegram.org/bot"

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: telegramAPIBase, // Default
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetBaseURL for testing
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

type InlineKeyboardButton struct {
	Text                         string      `json:"text"`
	URL                          string      `json:"url,omitempty"`
	CallbackData                 string      `json:"callback_data,omitempty"`
	WebApp                       *WebAppInfo `json:"web_app,omitempty"`
	SwitchInlineQuery            string      `json:"switch_inline_query,omitempty"`
	SwitchInlineQueryCurrentChat string      `json:"switch_inline_query_current_chat,omitempty"`
}

type KeyboardButton struct {
	Text string `json:"text"`
}

type WebAppInfo struct {
	URL string `json:"url"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`
	OneTimeKeyboard bool               `json:"one_time_keyboard"`
}

type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type CommandScope struct {
	Type string `json:"type"`
}

const (
	CommandScopeDefault         = "default"
	CommandScopeAllPrivateChats = "all_private_chats"
	CommandScopeAllGroupChats   = "all_group_chats"
)

type sendMessageReq struct {
	ChatID           int64       `json:"chat_id"`
	Text             string      `json:"text"`
	ParseMode        string      `json:"parse_mode,omitempty"`
	ReplyMarkup      interface{} `json:"reply_markup,omitempty"`
	ReplyToMessageID int64       `json:"reply_to_message_id,omitempty"`
	MessageThreadID  int64       `json:"message_thread_id,omitempty"`
}

func (c *Client) SendMessage(chatID int64, text string) error {
	return c.SendMessageWithReplyAndThread(chatID, text, 0, 0)
}

func (c *Client) SendMessageWithReply(chatID int64, text string, replyToID int64) error {
	return c.SendMessageWithReplyAndThread(chatID, text, replyToID, 0)
}

func (c *Client) SendMessageWithReplyAndThread(chatID int64, text string, replyToID int64, threadID int64) error {
	return c.sendJSON("sendMessage", sendMessageReq{
		ChatID:           chatID,
		Text:             text,
		ParseMode:        "HTML",
		ReplyToMessageID: replyToID,
		MessageThreadID:  threadID,
	})
}

// SendMessageWithButtons sends a message with inline keyboard buttons
func (c *Client) SendMessageWithButtons(chatID int64, text string, markup InlineKeyboardMarkup) error {
	return c.sendJSON("sendMessage", sendMessageReq{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})
}

func (c *Client) SendMessageWithMarkup(chatID int64, text string, markup interface{}) error {
	return c.SendMessageWithMarkupAndReplyAndThread(chatID, text, markup, 0, 0)
}

func (c *Client) SendMessageWithMarkupAndReply(chatID int64, text string, markup interface{}, replyToID int64) error {
	return c.SendMessageWithMarkupAndReplyAndThread(chatID, text, markup, replyToID, 0)
}

func (c *Client) SendMessageWithMarkupAndReplyAndThread(chatID int64, text string, markup interface{}, replyToID int64, threadID int64) error {
	return c.sendJSON("sendMessage", sendMessageReq{
		ChatID:           chatID,
		Text:             text,
		ParseMode:        "HTML",
		ReplyMarkup:      markup,
		ReplyToMessageID: replyToID,
		MessageThreadID:  threadID,
	})
}

type setCommandsReq struct {
	Commands     []BotCommand  `json:"commands"`
	Scope        *CommandScope `json:"scope,omitempty"`
	LanguageCode string        `json:"language_code,omitempty"`
}

type SetMyCommandsRequest struct {
	Commands     []BotCommand
	Scope        *CommandScope
	LanguageCode string
}

func (c *Client) SetMyCommands(req SetMyCommandsRequest) error {
	if len(req.Commands) == 0 {
		return nil
	}
	payload := setCommandsReq{
		Commands:     req.Commands,
		Scope:        req.Scope,
		LanguageCode: req.LanguageCode,
	}
	return c.sendJSON("setMyCommands", payload)
}

// InlineQuery types
type InlineQueryResultArticle struct {
	Type                string                `json:"type"`
	ID                  string                `json:"id"`
	Title               string                `json:"title"`
	InputMessageContent InputMessageContent   `json:"input_message_content"`
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
	Description         string                `json:"description,omitempty"`
}

type InputMessageContent struct {
	MessageText string `json:"message_text"`
	ParseMode   string `json:"parse_mode"`
}

type answerInlineQueryReq struct {
	InlineQueryID string      `json:"inline_query_id"`
	Results       interface{} `json:"results"`
	CacheTime     int         `json:"cache_time"`
	IsPersonal    bool        `json:"is_personal"`
}

func (c *Client) AnswerInlineQuery(queryID string, results interface{}) error {
	return c.sendJSON("answerInlineQuery", answerInlineQueryReq{
		InlineQueryID: queryID,
		Results:       results,
		CacheTime:     0,
		IsPersonal:    true,
	})
}

// CallbackQuery Answer
type answerCallbackQueryReq struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

func (c *Client) AnswerCallbackQuery(queryID string, text string) error {
	return c.sendJSON("answerCallbackQuery", answerCallbackQueryReq{
		CallbackQueryID: queryID,
		Text:            text,
	})
}

// EditMessageText
type editMessageTextReq struct {
	ChatID          int64                 `json:"chat_id,omitempty"`
	MessageID       int                   `json:"message_id,omitempty"`
	InlineMessageID string                `json:"inline_message_id,omitempty"`
	Text            string                `json:"text"`
	ParseMode       string                `json:"parse_mode"`
	ReplyMarkup     *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (c *Client) EditMessageText(inlineMessageID string, text string, markup *InlineKeyboardMarkup) error {
	return c.sendJSON("editMessageText", editMessageTextReq{
		InlineMessageID: inlineMessageID,
		Text:            text,
		ParseMode:       "HTML",
		ReplyMarkup:     markup,
	})
}

func (c *Client) sendJSON(method string, payload interface{}) error {
	url := fmt.Sprintf("%s%s/%s", c.baseURL, c.token, method)

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Debug: log the JSON being sent
	fmt.Printf("[DEBUG] Telegram API %s request: %s\n", method, string(body))

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	// Debug: log the response
	fmt.Printf("[DEBUG] Telegram API %s response (Status %d): %s\n", method, resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		bodyPreview := strings.TrimSpace(string(respBody))
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "…"
		}
		return fmt.Errorf("telegram api error: status %d body: %s", resp.StatusCode, bodyPreview)
	}

	return nil
}
