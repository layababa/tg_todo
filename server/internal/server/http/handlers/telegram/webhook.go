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

	"github.com/layababa/tg_todo/server/internal/models"
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
	userRepo     repository.UserRepository
	taskCreator  *task.Creator
	taskService  *task.Service // Added TaskService
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
	UserRepo     repository.UserRepository
	TaskCreator  *task.Creator
	TaskService  *task.Service // Added TaskService
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
		userRepo:     cfg.UserRepo,
		taskCreator:  cfg.TaskCreator,
		taskService:  cfg.TaskService, // Added TaskService
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
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
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
	InlineQuery   *InlineQuery   `json:"inline_query"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type InlineQuery struct {
	ID       string `json:"id"`
	From     User   `json:"from"`
	Query    string `json:"query"`
	Offset   string `json:"offset"`
	ChatType string `json:"chat_type"`
}

type CallbackQuery struct {
	ID              string   `json:"id"`
	From            User     `json:"from"`
	Message         *Message `json:"message,omitempty"`
	InlineMessageID string   `json:"inline_message_id,omitempty"`
	Data            string   `json:"data"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
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
				// Send welcome message - focus on core task management
				welcomeText := fmt.Sprintf(
					"👋 欢迎使用 Telegram To-Do 助手！\n\n"+
						"📝 **如何创建任务**\n"+
						"• 在群内 @%s + 文本\n"+
						"• 回复消息 + @%s\n"+
						"• 使用 @ 提及成员可指派任务\n\n"+
						"💡 输入 /help 查看更多功能",
					h.botUsername, h.botUsername,
				)

				// Try to add bind button if webAppURL is configured
				startParam := "bind_" + groupID
				markup := h.buildWebAppMarkup("⚙️ 高级设置", startParam)

				if markup != nil {
					// If webAppURL is configured, mention advanced features
					welcomeText = fmt.Sprintf(
						"👋 欢迎使用 Telegram To-Do 助手！\n\n"+
							"📝 **如何创建任务**\n"+
							"• 在群内 @%s + 文本\n"+
							"• 回复消息 + @%s\n"+
							"• 使用 @ 提及成员可指派任务\n\n"+
							"💡 点击下方按钮可配置高级功能（如 Notion 同步）",
						h.botUsername, h.botUsername,
					)
				}

				h.sendMessage(mcm.Chat.ID, welcomeText, markup)
			}
		} else if status == "left" || status == "kicked" {
			// Bot left
			groupID := fmt.Sprintf("%d", mcm.Chat.ID)
			h.groupService.UpdateStatus(ctx, groupID, "Inactive")
		}
	}

	// C. Inline Query
	if update.InlineQuery != nil {
		h.handleInlineQuery(ctx, update.InlineQuery)
		c.Status(http.StatusOK)
		return
	}

	// D. Callback Query
	if update.CallbackQuery != nil {
		h.handleCallbackQuery(ctx, update.CallbackQuery)
		c.Status(http.StatusOK)
		return
	}

	// B. Message (Existing logic)
	if update.Message != nil {
		msg := update.Message

		// Ensure user exists on first interaction
		h.ensureUser(ctx, msg)

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
			// PRD Story S1/S2: 群聊中 @Bot 或 Reply + @Bot 创建任务
			if h.shouldCreateTask(msg) {
				h.handleTaskCommand(ctx, msg)
			}
		}
	}

	c.Status(http.StatusOK)
}

// define structs locally for InlineQuery/CallbackQuery if not present in Update struct
// Based on previous file view, Update struct was:
/*
type Update struct {
	UpdateID     int64    `json:"update_id"`
	Message      *Message `json:"message"`
	MyChatMember *struct { ... }
}
*/
// I need to add InlineQuery and CallbackQuery fields to Update struct in this file first?
// Wait, I cannot add fields here inside a method replacement.
// I should rely on the previous tool call which edited `Update` struct?
// No, I haven't edited `Update` struct yet. I missed that step. I only planned to do it.
// I will edit `Update` struct in a separate replacement chunk in this tool call or a following one.
// Let's assume I will do it.

func (h *Handler) handleInlineQuery(ctx context.Context, iq *InlineQuery) {
	query := strings.TrimSpace(iq.Query)
	// format: assign <TaskID>
	if !strings.HasPrefix(query, "assign ") {
		return
	}
	taskID := strings.TrimPrefix(query, "assign ")
	if taskID == "" {
		return
	}

	// Fetch Task
	t, err := h.taskService.GetTask(ctx, taskID)
	if err != nil {
		h.logger.Error("handleInlineQuery: GetTask failed", zap.String("task_id", taskID), zap.Error(err))
		return // Ignore errors
	}
	h.logger.Info("handleInlineQuery: Task found", zap.String("task_id", t.ID), zap.String("title", t.Title))

	// Construct Result

	// Create Buttons
	var rows [][]telegram.InlineKeyboardButton

	// Row 1: Accept Button
	rows = append(rows, []telegram.InlineKeyboardButton{
		{
			Text:         "🙋‍♂️ 我来认领 (Claim)",
			CallbackData: fmt.Sprintf("accept_task:%s", t.ID),
		},
	})

	// Row 2: View Details (if WebApp URL available)
	if h.botUsername != "" {
		// Use "task" alias as configured by user
		appLink := fmt.Sprintf("https://t.me/%s/task?startapp=task_%s", h.botUsername, t.ID)
		rows = append(rows, []telegram.InlineKeyboardButton{
			{
				Text: "📋 查看详情",
				URL:  appLink,
			},
		})
	}

	markup := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}

	assigneeName := "待认领"
	if len(t.Assignees) > 0 {
		assigneeName = t.Assignees[0].Name
	}

	dueDate := "无"
	if t.DueAt != nil {
		dueDate = t.DueAt.Format("2006-01-02 15:04")
	}

	msgText := fmt.Sprintf(
		"📋 <b>任务分享</b>\n\n"+
			"<b>%s</b>\n"+
			"──────────────\n"+
			"👤 负责人: %s\n"+
			"📅 截止: %s\n"+
			"──────────────\n"+
			"👇 点击下方按钮认领或查看详情",
		t.Title, assigneeName, dueDate,
	)

	article := telegram.InlineQueryResultArticle{
		Type:        "article",
		ID:          taskID,
		Title:       fmt.Sprintf("分享任务: %s", t.Title),
		Description: fmt.Sprintf("当前负责人: %s", assigneeName),
		InputMessageContent: telegram.InputMessageContent{
			MessageText: msgText,
			ParseMode:   "HTML",
		},
		ReplyMarkup: markup,
	}

	if err := h.tgClient.AnswerInlineQuery(iq.ID, []telegram.InlineQueryResultArticle{article}); err != nil {
		h.logger.Error("failed to answer inline query", zap.Error(err))
	}
}

func (h *Handler) handleCallbackQuery(ctx context.Context, cq *CallbackQuery) {
	data := cq.Data
	// format: accept_task:<TaskID>
	if strings.HasPrefix(data, "accept_task:") {
		taskID := strings.TrimPrefix(data, "accept_task:")

		// Prepare User Model
		user := &models.User{
			TgID:       cq.From.ID,
			TgUsername: cq.From.Username,
			Name:       strings.TrimSpace(cq.From.FirstName + " " + cq.From.LastName),
		}

		// Assign Task via Service (handles user creation if needed)
		err := h.taskService.AssignTaskToTelegramUser(ctx, taskID, user)
		if err != nil {
			h.logger.Error("failed to assign task", zap.Error(err))
			h.tgClient.AnswerCallbackQuery(cq.ID, "❌ Failed to assign task")
			return
		}

		// Update success
		claimantName := cq.From.FirstName
		if cq.From.LastName != "" {
			claimantName += " " + cq.From.LastName
		}

		// Edit message
		t, _ := h.taskService.GetTask(ctx, taskID) // Fetch fresh logic
		title := "Task"
		if t != nil {
			title = t.Title
		}

		newText := fmt.Sprintf("📋 <b>Task: %s</b>\n\n✅ Assigned to %s", title, claimantName)

		// Create Success Buttons
		var rows [][]telegram.InlineKeyboardButton
		if h.botUsername != "" {
			// Button 1: View Details
			// User has configured "task" alias for Direct Link
			appLink := fmt.Sprintf("https://t.me/%s/task?startapp=task_%s", h.botUsername, t.ID)

			// Button 2: View All Todos (Home)
			// User has configured "home" alias for Direct Link
			homeLink := fmt.Sprintf("https://t.me/%s/home", h.botUsername)

			rows = append(rows, []telegram.InlineKeyboardButton{
				{Text: "📋 查看任务详情", URL: appLink},
				{Text: "🏠 查看所有待办", URL: homeLink},
			})
		}

		successMarkup := &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
		h.tgClient.EditMessageText(cq.InlineMessageID, newText, successMarkup)
		h.tgClient.AnswerCallbackQuery(cq.ID, "✅ You are now the assignee!")
	}
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
		ChatTitle: msg.Chat.Title,
		ChatType:  msg.Chat.Type,
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

	// Build detailed reply message
	var replyText string
	assigneeCount := len(createdTask.Assignees)

	// Build task URL for Mini App using Telegram deep link
	// Format: https://t.me/<BotUsername>?startapp=task_<TaskID>
	taskURL := ""
	if h.botUsername != "" {
		// Remove @ prefix if present
		cleanBotName := strings.TrimPrefix(h.botUsername, "@")
		taskURL = fmt.Sprintf("https://t.me/%s?startapp=task_%s", cleanBotName, createdTask.ID)
	}

	// Check if this is a group chat
	// Telegram WebApp buttons are NOT supported in group chats, only in private chats
	isGroupChat := msg.Chat.Type == "group" || msg.Chat.Type == "supergroup"

	if isGroupChat {
		// In group chats: @ assignees and provide task URL
		if assigneeCount > 0 {
			// @ all assignees
			var mentions []string
			for _, assignee := range createdTask.Assignees {
				if assignee.TgUsername != "" {
					mentions = append(mentions, "@"+assignee.TgUsername)
				}
			}

			if len(mentions) > 0 {
				// Use HTML link format: <a href="URL">text</a>
				replyText = fmt.Sprintf("✅ 已创建任务：%s\n\n%s 请点击 <a href=\"%s\">查看任务</a>",
					createdTask.Title,
					strings.Join(mentions, " "),
					taskURL)
			} else {
				// No usernames available, just show task created
				replyText = fmt.Sprintf("✅ 已创建任务：%s\n\n👥 已指派给 %d 人\n<a href=\"%s\">查看任务</a>",
					createdTask.Title,
					assigneeCount,
					taskURL)
			}
		} else {
			// No assignees
			replyText = fmt.Sprintf("✅ 已创建任务：%s\n\n<a href=\"%s\">查看任务</a>", createdTask.Title, taskURL)
		}
	} else {
		// In private chats: use WebApp buttons
		if assigneeCount > 1 {
			replyText = fmt.Sprintf("✅ 已创建任务：%s\n👥 已指派给 %d 人", createdTask.Title, assigneeCount)
		} else {
			replyText = fmt.Sprintf("✅ 已创建任务：%s", createdTask.Title)
		}
	}

	var markup interface{}
	if isGroupChat {
		// No buttons in group chats
		markup = nil
	} else {
		// In private chats, we can use WebApp buttons
		if createdTask.DatabaseID == nil {
			groupID := fmt.Sprintf("%d", msg.Chat.ID)
			startParam := "bind_" + groupID
			markup = h.buildWebAppMarkup("⚙️ 设置", startParam)
		} else {
			replyText += "\n✓ 已同步"
			taskParam := fmt.Sprintf("task_%s", createdTask.ID)
			markup = h.buildWebAppMarkup("📋 查看详情", taskParam)
		}
	}

	h.sendMessage(msg.Chat.ID, replyText, markup)
}

func (h *Handler) sendMessage(chatID int64, text string, markup interface{}) {
	h.logger.Debug("sendMessage called",
		zap.Int64("chatID", chatID),
		zap.Bool("hasMarkup", markup != nil))
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
	h.logger.Debug("buildWebAppMarkup called",
		zap.String("webAppURL", h.webAppURL),
		zap.String("buttonText", buttonText),
		zap.String("startParam", startParam),
		zap.String("generatedURL", url))
	if url == "" {
		h.logger.Warn("buildWebAppMarkup returning nil because URL is empty")
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

// shouldCreateTask checks if a message should trigger task creation
// According to PRD Story S1/S2:
// - Group chat: @Bot + text creates task
// - Group chat: Reply + @Bot creates task
// - Private chat: Any non-command text creates task
func (h *Handler) shouldCreateTask(msg *Message) bool {
	if msg == nil {
		return false
	}

	text := msg.Text
	if text == "" {
		return false
	}

	// Private chat: any non-command text creates a task
	if msg.Chat.Type == "private" {
		// Commands are handled separately, so if we reach here it's not a command
		return true
	}

	// Group chats: only create task if bot is mentioned
	if msg.Chat.Type != "group" && msg.Chat.Type != "supergroup" {
		return false
	}

	// Check if bot is mentioned
	botMentioned := false
	if h.botUsername != "" {
		botMentioned = strings.Contains(text, "@"+h.botUsername)
	}

	// Case 1: @Bot + text (direct mention)
	if botMentioned {
		return true
	}

	// Case 2: Reply + @ (any mention in reply)
	if msg.ReplyToMessage != nil && strings.Contains(text, "@") {
		return true
	}

	return false
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

// ensureUser creates a user record if it doesn't exist when they interact with the bot
func (h *Handler) ensureUser(ctx context.Context, msg *Message) {
	if h.userRepo == nil || msg == nil || msg.From.ID == 0 {
		return
	}

	// Check if user exists
	_, err := h.userRepo.FindByTgID(ctx, msg.From.ID)
	if err == nil {
		// User exists, nothing to do
		return
	}

	// Build user name
	name := msg.From.FirstName
	if msg.From.LastName != "" {
		name += " " + msg.From.LastName
	}
	if name == "" {
		name = msg.From.Username
	}
	if name == "" {
		name = "User"
	}

	// Create new user
	newUser := &models.User{
		TgID:       msg.From.ID,
		Name:       name,
		TgUsername: msg.From.Username,
	}

	if err := h.userRepo.Create(ctx, newUser); err != nil {
		h.logger.Warn("failed to create user on first interaction", zap.Error(err), zap.Int64("tg_id", msg.From.ID))
	} else {
		h.logger.Info("auto-created user on first interaction", zap.Int64("tg_id", msg.From.ID), zap.String("name", name))
	}
}
