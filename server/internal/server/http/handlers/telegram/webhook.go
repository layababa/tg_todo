package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	"github.com/layababa/tg_todo/server/internal/repository"
	groupsvc "github.com/layababa/tg_todo/server/internal/service/group"
	"github.com/layababa/tg_todo/server/internal/service/task"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
)

const (
	HeaderBotApiSecretToken = "X-Telegram-Bot-Api-Secret-Token"
)

// Handler handles telegram webhook requests
type Handler struct {
	logger       *zap.Logger
	deduplicator telegram.Deduplicator
	repo         repository.TelegramUpdateRepository
	taskCreator  *task.Creator
	groupService *groupsvc.Service
	tgClient     *telegram.Client
	secretToken  string
	botUsername  string
	webAppURL    string
}

// Config holds configuration for the handler
type Config struct {
	Logger       *zap.Logger
	Deduplicator telegram.Deduplicator
	Repo         repository.TelegramUpdateRepository
	TaskCreator  *task.Creator
	GroupService *groupsvc.Service
	TgClient     *telegram.Client
	SecretToken  string
	BotUsername  string
	WebAppURL    string
}

// NewHandler creates a new telegram webhook handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		logger:       cfg.Logger,
		deduplicator: cfg.Deduplicator,
		repo:         cfg.Repo,
		taskCreator:  cfg.TaskCreator,
		groupService: cfg.GroupService,
		tgClient:     cfg.TgClient,
		secretToken:  cfg.SecretToken,
		botUsername:  strings.TrimPrefix(cfg.BotUsername, "@"),
		webAppURL:    cfg.WebAppURL,
	}
}

type Message struct {
	MessageID int64 `json:"message_id"`
	From      struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	} `json:"from"`
	Chat struct {
		ID    int64  `json:"id"`
		Type  string `json:"type"` // private, group, supergroup
		Title string `json:"title"`
	} `json:"chat"`
	Text           string `json:"text"`
	ReplyToMessage *struct {
		MessageID int64 `json:"message_id"`
	} `json:"reply_to_message"`
	ForwardDate int64 `json:"forward_date"`
	ForwardFrom *struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"forward_from"`
	ForwardFromChat *struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Type  string `json:"type"`
	} `json:"forward_from_chat"`
}

// Update represents the basic structure we need to extract update_id and content
type Update struct {
	UpdateID     int64    `json:"update_id"`
	Message      *Message `json:"message"`
	MyChatMember *struct {
		Chat struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
		} `json:"chat"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		NewChatMember struct {
			Status string `json:"status"` // member, administrator, kicked, left
			User   struct {
				ID       int64  `json:"id"`
				IsBot    bool   `json:"is_bot"`
				Username string `json:"username"`
			} `json:"user"`
		} `json:"new_chat_member"`
	} `json:"my_chat_member"`
}

// HandleWebhook processes incoming webhook requests
func (h *Handler) HandleWebhook(c *gin.Context) {
	// ... (Validate Secret Token, Read Body - same as before) ...
	// 1. Validate Secret Token
	if h.secretToken != "" {
		token := c.GetHeader(HeaderBotApiSecretToken)
		if token != h.secretToken {
			h.logger.Warn("invalid secret token", zap.String("token", token))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}

	// 2. Read Body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read body", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 3. Parse Update
	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		h.logger.Error("failed to unmarshal update", zap.Error(err))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 4. Deduplicate
	isDuplicate, err := h.deduplicator.IsDuplicate(c.Request.Context(), update.UpdateID)
	if err != nil {
		h.logger.Error("failed to check duplicate", zap.Error(err))
		// Continue even if fail? Or abort. Safe to abort.
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if isDuplicate {
		h.logger.Info("ignoring duplicate update", zap.Int64("update_id", update.UpdateID))
		c.Status(http.StatusOK)
		return
	}

	// 5. Save Raw Update (Async potentially, but here sync is fine)
	rawUpdate := &repository.TelegramUpdate{
		UpdateID: update.UpdateID,
		RawData:  datatypes.JSON(body),
	}
	if err := h.repo.Save(c.Request.Context(), rawUpdate); err != nil {
		h.logger.Error("failed to save update", zap.Error(err))
		// We still process logic even if save fails? Maybe best effort.
	}

	// 6. Handle Logic
	ctx := c.Request.Context()

	// A. MyChatMember (Bot added/removed)
	if update.MyChatMember != nil {
		mcm := update.MyChatMember
		// Check if update is about THIS bot
		// Usually my_chat_member updates contain the user whose status changed.
		// If implementation requires checking bot ID, we assume it's correct context.
		// Check status
		status := mcm.NewChatMember.Status
		if status == "member" || status == "administrator" {
			// Bot joined
			adminID := fmt.Sprintf("%d", mcm.From.ID)
			groupID := fmt.Sprintf("%d", mcm.Chat.ID)
			err := h.groupService.EnsureGroup(ctx, groupID, mcm.Chat.Title, adminID)
			if err != nil {
				h.logger.Error("failed to ensure group", zap.Error(err))
			} else {
				h.tgClient.SendMessage(mcm.Chat.ID, "Hello! I am ready. Use /bind to connect Notion.")
			}
		} else if status == "left" || status == "kicked" {
			// Bot left
			groupID := fmt.Sprintf("%d", mcm.Chat.ID)
			h.groupService.UpdateStatus(ctx, groupID, "Inactive")
		}
	}

	// B. Message
	if update.Message != nil {
		msg := update.Message

		// Check for forward first
		if msg.ForwardDate > 0 || msg.ForwardFrom != nil || msg.ForwardFromChat != nil {
			h.handleForwardedMessage(ctx, msg)
			c.Status(http.StatusOK)
			return
		}

		cmd, args := extractCommand(msg.Text)
		switch cmd {
		case "/start":
			h.handleStart(ctx, msg.Chat.ID, args)
		case "/help":
			h.handleHelp(msg.Chat.ID)
		case "/settings":
			h.handleSettings(msg.Chat.ID, msg.Chat.Type)
		case "/bind":
			h.handleBind(ctx, msg.Chat.ID, msg.From.ID, msg.Chat.Title)
		case "/todo":
			h.handleTaskCommand(ctx, msg)
		case "/menu":
			h.handleMenu(msg.Chat.ID)
		case "/close", "/hide":
			h.handleHideKeyboard(msg.Chat.ID)
		default:
			if msg.ReplyToMessage != nil && strings.Contains(msg.Text, "@") {
				h.handleTaskCommand(ctx, msg)
			}
		}
	}

	c.Status(http.StatusOK)
}

func (h *Handler) handleStart(ctx context.Context, chatID int64, args []string) {
	var startParam string
	if len(args) > 0 {
		startParam = args[0]
	}
	openAppMarkup := h.buildWebAppMarkup("打开 Mini App", startParam)
	text := "👋 欢迎使用 Telegram To-Do 助手！\n\n• 直接输入 /todo 或引用消息即可把任务保存到内置数据库\n• 随时打开 Mini App 管理我的待办、群组与设置\n• 需要同步 Notion 时再进入设置绑定即可\n• 输入 /help 查看所有指令与操作示例"
	if link := h.resolveShareableLink(startParam); link != "" {
		text += fmt.Sprintf("\n\n🔗 直接打开：%s", link)
	}
	h.sendMessage(chatID, text, openAppMarkup)

	quickActions := "⚡️ 快捷操作：\n" +
		"• 点 /todo 直接创建任务\n" +
		"• 点 /settings 设置默认数据库\n" +
		"• 点 /help 查看全部指令"
	h.sendMessage(chatID, quickActions, h.buildQuickCommandKeyboard())
}

func (h *Handler) handleHelp(chatID int64) {
	text := "🆘 指令清单：\n" +
		"/start — 开始使用 / 打开 Mini App\n" +
		"/menu — 展示快捷菜单（/todo、/settings 等）\n" +
		"/close — 隐藏快捷菜单\n" +
		"/help — 查看帮助与功能演示\n" +
		"/settings — 打开个人设置（绑定 Notion、默认数据库）\n" +
		"/bind — (群管理员) 绑定当前群的 Notion 数据库\n" +
		"/todo — (群聊) 快速创建任务，或引用消息后 @Bot 生成任务\n\n" +
		"更多使用说明：Mini App > 帮助中心。"
	h.sendMessage(chatID, text, h.buildHelpInlineMarkup())
}

func (h *Handler) handleSettings(chatID int64, chatType string) {
	if chatType != "private" {
		h.sendMessage(chatID, "⚠️ 请在与机器人私聊中输入 /settings，以免泄露个人设置。", nil)
		return
	}
	const startParam = "settings"
	text := "🔧 打开 Mini App，配置个人设置、默认数据库与时区。"
	markup := h.buildWebAppMarkup("打开个人设置", startParam)
	if link := h.resolveShareableLink(startParam); link != "" {
		text += fmt.Sprintf("\n\n🔗 直接打开：%s", link)
	}
	h.sendMessage(chatID, text, markup)
}

func (h *Handler) handleBind(ctx context.Context, chatID, userID int64, title string) {
	if h.groupService != nil {
		groupID := fmt.Sprintf("%d", chatID)
		if err := h.groupService.EnsureGroup(ctx, groupID, title, fmt.Sprintf("%d", userID)); err != nil {
			h.logger.Error("failed to ensure group on /bind", zap.Error(err))
		}
	}
	groupID := fmt.Sprintf("%d", chatID)
	startParam := "bind_" + groupID
	text := fmt.Sprintf("为群组「%s」绑定 Notion Database，完成后即可直接在群内引用消息创建任务。", title)
	markup := h.buildWebAppMarkup("绑定 Notion 数据库", startParam)
	if link := h.resolveShareableLink(startParam); link != "" {
		text += fmt.Sprintf("\n\n🔗 直接打开：%s", link)
	}
	h.sendMessage(chatID, text, markup)
}

func (h *Handler) handleTaskCommand(ctx context.Context, msg *Message) {
	if h.taskCreator == nil || msg == nil {
		return
	}
	input := task.CreateInput{
		ChatID:    msg.Chat.ID,
		CreatorID: msg.From.ID,
		Text:      msg.Text,
	}
	if msg.ReplyToMessage != nil {
		input.ReplyToID = msg.ReplyToMessage.MessageID
	}
	createdTask, err := h.taskCreator.CreateTask(ctx, input)
	if err != nil {
		h.logger.Error("failed to create task", zap.Error(err))
		h.sendMessage(msg.Chat.ID, "❌ 创建任务失败，请稍后再试。", nil)
		return
	}
	var markup interface{}
	replyText := fmt.Sprintf("✅ 已创建任务：%s", createdTask.Title)

	if createdTask.DatabaseID == nil {
		replyText += "\n(当前仅保存在服务端，待绑定 Notion 后可同步)"
		// Add Bind Button
		groupID := fmt.Sprintf("%d", msg.Chat.ID)
		startParam := "bind_" + groupID
		markup = h.buildWebAppMarkup("⚙️ 绑定 Notion", startParam)
	} else {
		replyText += "\n(已同步到 Notion)"
	}
	h.sendMessage(msg.Chat.ID, replyText, markup)
}

func (h *Handler) sendMessage(chatID int64, text string, markup interface{}) {
	var err error
	if markup != nil {
		err = h.tgClient.SendMessageWithMarkup(chatID, text, markup)
	} else {
		err = h.tgClient.SendMessage(chatID, text)
	}
	if err != nil {
		h.logger.Error("failed to send telegram message", zap.Error(err), zap.Int64("chat_id", chatID))
	}
}

func (h *Handler) buildWebAppMarkup(buttonText, startParam string) *telegram.InlineKeyboardMarkup {
	url := h.buildWebAppButtonURL(startParam)
	if url == "" {
		return nil
	}
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: buttonText,
					WebApp: &telegram.WebAppInfo{
						URL: url,
					},
				},
			},
		},
	}
}

func (h *Handler) buildWebAppButtonURL(startParam string) string {
	if h.webAppURL == "" {
		return ""
	}
	parsed, err := url.Parse(h.webAppURL)
	if err != nil {
		h.logger.Warn("invalid web app url", zap.String("url", h.webAppURL), zap.Error(err))
		return ""
	}
	startParam = strings.TrimSpace(startParam)
	if startParam != "" {
		q := parsed.Query()
		q.Set("tg_web_app_start_param", startParam)
		parsed.RawQuery = q.Encode()
	}
	return parsed.String()
}

func (h *Handler) buildStartAppURL(startParam string) string {
	param := strings.TrimSpace(startParam)
	if h.botUsername != "" {
		base := fmt.Sprintf("https://t.me/%s/app", h.botUsername)
		if param != "" {
			base = fmt.Sprintf("%s?startapp=%s", base, url.QueryEscape(param))
		}
		return base
	}
	if h.webAppURL != "" {
		base := strings.TrimRight(h.webAppURL, "/")
		if param != "" {
			sep := "?"
			if strings.Contains(base, "?") {
				sep = "&"
			}
			base = fmt.Sprintf("%s%stg_web_app_start_param=%s", base, sep, url.QueryEscape(param))
		}
		return base
	}
	return ""
}

func (h *Handler) resolveShareableLink(startParam string) string {
	if link := h.buildStartAppURL(startParam); link != "" {
		return link
	}
	return h.buildWebAppButtonURL(startParam)
}

func (h *Handler) buildQuickCommandKeyboard() *telegram.ReplyKeyboardMarkup {
	return &telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.KeyboardButton{
			{
				{Text: "/todo"},
				{Text: "/settings"},
			},
			{
				{Text: "/help"},
				{Text: "/close"},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
}

func (h *Handler) buildHelpInlineMarkup() *telegram.InlineKeyboardMarkup {
	openAppURL := h.buildWebAppButtonURL("")
	rows := [][]telegram.InlineKeyboardButton{}
	if openAppURL != "" {
		rows = append(rows, []telegram.InlineKeyboardButton{
			{
				Text: "打开 Mini App",
				WebApp: &telegram.WebAppInfo{
					URL: openAppURL,
				},
			},
		})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{
			Text:                         "快捷输入 /todo",
			SwitchInlineQueryCurrentChat: "/todo ",
		},
		{
			Text:                         "输入 /menu",
			SwitchInlineQueryCurrentChat: "/menu",
		},
	})
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func (h *Handler) handleMenu(chatID int64) {
	h.sendMessage(chatID, "📋 已为您展示快捷菜单，直接点按钮即可发送指令。输入 /close 可以在任意时刻隐藏。", h.buildQuickCommandKeyboard())
}

func (h *Handler) handleHideKeyboard(chatID int64) {
	h.sendMessage(chatID, "✅ 已隐藏快捷菜单，如需再次显示请输入 /menu。", &telegram.ReplyKeyboardRemove{RemoveKeyboard: true})
}

func extractCommand(text string) (string, []string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return "", nil
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", nil
	}
	cmd := strings.ToLower(parts[0])
	if idx := strings.Index(cmd, "@"); idx >= 0 {
		cmd = cmd[:idx]
	}
	return cmd, parts[1:]
}

func (h *Handler) handleForwardedMessage(ctx context.Context, msg *Message) {
	if h.taskCreator == nil {
		return
	}

	// Prepare metadata
	meta := make(map[string]interface{})
	sourceName := "Unknown"

	if msg.ForwardFrom != nil {
		sourceName = msg.ForwardFrom.FirstName
		if msg.ForwardFrom.LastName != "" {
			sourceName += " " + msg.ForwardFrom.LastName
		}
		if msg.ForwardFrom.Username != "" {
			sourceName += " (@" + msg.ForwardFrom.Username + ")"
		}
	} else if msg.ForwardFromChat != nil {
		sourceName = msg.ForwardFromChat.Title
	} else {
		sourceName = "Anonymous Forward"
	}
	meta["source"] = sourceName

	text := msg.Text
	if text == "" {
		h.sendMessage(msg.Chat.ID, "⚠️ 暂不支持转发非文本消息。", nil)
		return
	}

	input := task.CreateInput{
		ChatID:    msg.Chat.ID,
		CreatorID: msg.From.ID,
		Text:      text,
		ReplyToID: 0,
	}

	createdTask, err := h.taskCreator.CreatePersonalTask(ctx, input, meta)
	if err != nil {
		h.logger.Error("failed to create personal task", zap.Error(err))
		h.sendMessage(msg.Chat.ID, "❌ 保存任务失败，请稍后再试。", nil)
		return
	}

	var markup interface{}
	replyText := fmt.Sprintf("✅ 已保存到收件箱：%s", createdTask.Title)

	if createdTask.DatabaseID == nil {
		replyText += "\n(仅保存在本地，建议绑定 Notion 以开启自动同步)"
		// Add Settings Button for private chat
		markup = h.buildWebAppMarkup("⚙️ 去绑定", "settings")
	} else {
		replyText += "\n(已同步到 Notion)"
	}
	h.sendMessage(msg.Chat.ID, replyText, markup)
}
