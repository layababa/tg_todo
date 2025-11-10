package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zz/tg_todo/server/internal/task"
)

var (
	errSkipTask   = errors.New("skip task creation")
	errEmptyTitle = errors.New("empty task title")
)

// Bot 负责与 Telegram API 通讯并调用任务服务。
type Bot struct {
	token       string
	apiBase     string
	client      *http.Client
	taskService *task.Service
	pollTimeout time.Duration
	offset      int
	username    string
	memberCache map[int64]map[string]task.Person
	userCache   map[string]task.Person
}

// New 实例化 Bot。
func New(token, apiBase string, taskSvc *task.Service) *Bot {
	return &Bot{
		token:       token,
		apiBase:     strings.TrimRight(apiBase, "/"),
		client:      &http.Client{Timeout: 35 * time.Second},
		taskService: taskSvc,
		pollTimeout: 30 * time.Second,
		memberCache: make(map[int64]map[string]task.Person),
		userCache:   make(map[string]task.Person),
	}
}

// Start 启动长轮询，并确保可以识别 Bot 自身用户名以校验群聊 @。
func (b *Bot) Start(ctx context.Context) error {
	if b.token == "" {
		return fmt.Errorf("bot token is empty")
	}
	if err := b.ensureProfile(ctx); err != nil {
		return fmt.Errorf("bot: getMe failed: %w", err)
	}

	log.Println("bot: start polling Telegram updates")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := b.fetchUpdates(ctx)
		if err != nil {
			log.Printf("bot: fetch updates failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, update := range updates {
			b.offset = update.UpdateID + 1
			if err := b.handleUpdate(ctx, update); err != nil {
				log.Printf("bot: handle update failed: %v", err)
			}
		}
	}
}

func (b *Bot) fetchUpdates(ctx context.Context) ([]Update, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/bot%s/getUpdates?timeout=%d&offset=%d", b.apiBase, b.token, int(b.pollTimeout.Seconds()), b.offset),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram response not ok")
	}
	return result.Result, nil
}

func (b *Bot) handleUpdate(ctx context.Context, update Update) error {
	if update.Message == nil {
		return nil
	}
	text := strings.TrimSpace(update.Message.Text)
	switch {
	case strings.HasPrefix(text, "/tasks"):
		return b.replyWithTasks(ctx, update.Message.Chat.ID)
	case strings.HasPrefix(text, "/ping"):
		return b.sendMessage(ctx, update.Message.Chat.ID, "pong ok")
	default:
		if err := b.tryCreateTask(ctx, update.Message); err != nil && !errors.Is(err, errSkipTask) {
			return err
		}
		return nil
	}
}

func (b *Bot) replyWithTasks(ctx context.Context, chatID int64) error {
	tasks, err := b.taskService.List(ctx)
	if err != nil {
		return b.sendMessage(ctx, chatID, "获取任务失败，请稍后再试。")
	}
	if len(tasks) == 0 {
		return b.sendMessage(ctx, chatID, "暂无相关任务。")
	}
	var buf strings.Builder
	buf.WriteString("当前任务：\n")
	for _, task := range tasks {
		buf.WriteString(fmt.Sprintf("• %s [%s]\n", task.Title, task.Status))
	}
	return b.sendMessage(ctx, chatID, buf.String())
}

func (b *Bot) tryCreateTask(ctx context.Context, msg *Message) error {
	if !b.shouldHandleMessage(msg) {
		return errSkipTask
	}
	input, err := b.buildCreateInput(ctx, msg)
	if err != nil {
		switch {
		case errors.Is(err, errSkipTask):
			return errSkipTask
		case errors.Is(err, errEmptyTitle):
			return b.sendMessage(ctx, msg.Chat.ID, "请提供任务内容，比如 /todo 买牛奶。")
		default:
			return err
		}
	}

	created, err := b.taskService.Create(ctx, input)
	if err != nil {
		if errors.Is(err, task.ErrInvalidInput) {
			return b.sendMessage(ctx, msg.Chat.ID, "任务创建失败：字段缺失或指派信息无效。")
		}
		return err
	}
	return b.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("任务已创建（#%s）：%s", created.ID, created.Title))
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string) error {
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", b.apiBase, b.token)
	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	data.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendMessage failed: %s", resp.Status)
	}
	return nil
}

func (b *Bot) ensureProfile(ctx context.Context) error {
	if b.username != "" {
		return nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/bot%s/getMe", b.apiBase, b.token),
		nil,
	)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("telegram response not ok")
	}
	b.username = strings.TrimPrefix(result.Result.Username, "@")
	if b.username == "" {
		log.Println("bot: warning - username empty,群聊 @ 检测将不可用")
	}
	return nil
}

func (b *Bot) shouldHandleMessage(msg *Message) bool {
	if msg == nil || msg.From == nil {
		return false
	}
	if msg.From.IsBot {
		return false
	}
	if msg.Chat.Type == "private" {
		return true
	}
	if b.username == "" {
		return false
	}
	content := strings.ToLower(strings.TrimSpace(msg.Text + " " + msg.Caption))
	if content == "" {
		return false
	}
	return strings.Contains(content, "@"+strings.ToLower(b.username))
}

func (b *Bot) buildCreateInput(ctx context.Context, msg *Message) (task.CreateInput, error) {
	if msg.From == nil {
		return task.CreateInput{}, errSkipTask
	}

	commandTitle, isCommand := parseCommandTitle(msg.Text)
	if isCommand && strings.TrimSpace(commandTitle) == "" {
		return task.CreateInput{}, errEmptyTitle
	}

	var (
		title           string
		sourceChat      *Chat
		sourceMessageID int
		sourceURL       *string
	)

	if isCommand {
		title = commandTitle
	} else if reply := msg.ReplyToMessage; reply != nil {
		if reply.From != nil {
			b.cachePerson(msg.Chat.ID, userToPerson(reply.From))
		}
		if reply.ForwardFrom != nil {
			b.cachePerson(msg.Chat.ID, userToPerson(reply.ForwardFrom))
		}
		title = strings.TrimSpace(messageContent(reply))
		if title == "" {
			title = fmt.Sprintf("引用的消息 #%d", reply.MessageID)
		}
		replyChat := reply.Chat
		sourceChat = &replyChat
		sourceMessageID = reply.MessageID
		sourceURL = buildSourceURL(sourceChat, sourceMessageID)
	} else {
		if msg.ForwardFrom != nil {
			b.cachePerson(msg.Chat.ID, userToPerson(msg.ForwardFrom))
		}
		title = strings.TrimSpace(messageContent(msg))
		if msg.ForwardFromChat != nil && msg.ForwardFromMessageID != 0 {
			sourceChat = msg.ForwardFromChat
			sourceMessageID = msg.ForwardFromMessageID
			sourceURL = buildSourceURL(sourceChat, sourceMessageID)
		}
	}

	if strings.TrimSpace(title) == "" && sourceMessageID != 0 {
		title = fmt.Sprintf("来自消息 #%d", sourceMessageID)
	}
	if strings.TrimSpace(title) == "" {
		return task.CreateInput{}, errSkipTask
	}

	creator := userToPerson(msg.From)
	b.cachePerson(msg.Chat.ID, creator)
	assignees := b.collectAssignees(ctx, msg, creator)

	messageRef := &task.TelegramMessageRef{
		ChatID:    msg.Chat.ID,
		MessageID: int64(msg.MessageID),
	}
	if sourceChat != nil && sourceMessageID != 0 {
		messageRef.ChatID = sourceChat.ID
		messageRef.MessageID = int64(sourceMessageID)
	}

	return task.CreateInput{
		Title:            title,
		Creator:          creator,
		Assignees:        assignees,
		Status:           task.StatusPending,
		SourceMessageURL: sourceURL,
		TelegramMessage:  messageRef,
	}, nil
}

func parseCommandTitle(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return "", false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", false
	}
	cmd := strings.TrimPrefix(fields[0], "/")
	cmd = strings.ToLower(cmd)
	if idx := strings.Index(cmd, "@"); idx >= 0 {
		cmd = cmd[:idx]
	}
	switch cmd {
	case "todo", "task", "addtask":
		if len(fields) == 1 {
			return "", true
		}
		return strings.TrimSpace(strings.Join(fields[1:], " ")), true
	default:
		return "", false
	}
}

func messageContent(msg *Message) string {
	if msg == nil {
		return ""
	}
	if strings.TrimSpace(msg.Text) != "" {
		return strings.TrimSpace(msg.Text)
	}
	return strings.TrimSpace(msg.Caption)
}

func (b *Bot) collectAssignees(ctx context.Context, msg *Message, creator task.Person) []task.Person {
	result := []task.Person{creator}
	seen := map[string]struct{}{creator.ID: {}}

	addPerson := func(person task.Person) {
		if person.ID == "" {
			return
		}
		if _, ok := seen[person.ID]; ok {
			return
		}
		seen[person.ID] = struct{}{}
		result = append(result, person)
		b.cachePerson(msg.Chat.ID, person)
	}

	addUser := func(u *User) {
		if u == nil || u.IsBot {
			return
		}
		person := userToPerson(u)
		addPerson(person)
	}

	processEntities := func(entities []MessageEntity, content string) {
		for _, entity := range entities {
			switch entity.Type {
			case "text_mention":
				addUser(entity.User)
			case "mention":
				username := extractEntityUsername(content, entity)
				if username == "" {
					continue
				}
				person, err := b.resolveUsername(ctx, msg.Chat.ID, username)
				if err != nil {
					log.Printf("bot: resolve username @%s failed: %v", username, err)
					continue
				}
				addPerson(person)
			}
		}
	}

	processEntities(msg.Entities, msg.Text)
	processEntities(msg.CaptionEntities, msg.Caption)

	return result
}

func (b *Bot) resolveUsername(ctx context.Context, chatID int64, username string) (task.Person, error) {
	normalized := normalizeUsername(username)
	if normalized == "" {
		return task.Person{}, fmt.Errorf("empty username")
	}
	if person, ok := b.userCache[normalized]; ok {
		return person, nil
	}
	if chatCache, ok := b.memberCache[chatID]; ok {
		if person, ok := chatCache[normalized]; ok {
			return person, nil
		}
	}

	person, err := b.fetchChatMember(ctx, chatID, normalized)
	if err != nil {
		return task.Person{}, err
	}
	b.cachePerson(chatID, person)
	return person, nil
}

func (b *Bot) fetchChatMember(ctx context.Context, chatID int64, username string) (task.Person, error) {
	endpoint := fmt.Sprintf("%s/bot%s/getChatMember", b.apiBase, b.token)
	data := url.Values{}
	data.Set("chat_id", fmt.Sprintf("%d", chatID))
	queryUsername := username
	if !strings.HasPrefix(queryUsername, "@") {
		queryUsername = "@" + queryUsername
	}
	data.Set("user_id", queryUsername)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return task.Person{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return task.Person{}, err
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool       `json:"ok"`
		Result      ChatMember `json:"result"`
		Description string     `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return task.Person{}, err
	}
	if !result.OK {
		if result.Description != "" {
			return task.Person{}, fmt.Errorf("getChatMember failed: %s", result.Description)
		}
		return task.Person{}, fmt.Errorf("getChatMember failed")
	}
	if result.Result.User == nil {
		return task.Person{}, fmt.Errorf("getChatMember returned empty user")
	}
	return userToPerson(result.Result.User), nil
}

func (b *Bot) cachePerson(chatID int64, person task.Person) {
	if person.ID == "" {
		return
	}
	if person.Username != "" {
		username := normalizeUsername(person.Username)
		if username != "" {
			b.userCache[username] = person
			if _, ok := b.memberCache[chatID]; !ok {
				b.memberCache[chatID] = make(map[string]task.Person)
			}
			b.memberCache[chatID][username] = person
		}
	}
}

func extractEntityUsername(content string, entity MessageEntity) string {
	text := extractEntityText(content, entity)
	if text == "" {
		return ""
	}
	return normalizeUsername(text)
}

func extractEntityText(content string, entity MessageEntity) string {
	if content == "" || entity.Length <= 0 {
		return ""
	}
	runes := []rune(content)
	if entity.Offset < 0 || entity.Offset >= len(runes) {
		return ""
	}
	end := entity.Offset + entity.Length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[entity.Offset:end])
}

func normalizeUsername(name string) string {
	value := strings.TrimSpace(name)
	value = strings.TrimPrefix(value, "@")
	value = strings.TrimSpace(value)
	return strings.ToLower(value)
}

func userToPerson(u *User) task.Person {
	if u == nil {
		return task.Person{}
	}
	displayName := formatDisplayName(u)
	return task.Person{
		ID:          strconv.FormatInt(u.ID, 10),
		DisplayName: displayName,
		Username:    u.Username,
	}
}

func formatDisplayName(u *User) string {
	name := strings.TrimSpace(strings.Join([]string{u.FirstName, u.LastName}, " "))
	if name != "" {
		return name
	}
	if u.Username != "" {
		return u.Username
	}
	return strconv.FormatInt(u.ID, 10)
}

func buildSourceURL(chat *Chat, messageID int) *string {
	if chat == nil {
		return nil
	}
	username := strings.TrimPrefix(chat.Username, "@")
	if username == "" {
		switch chat.Type {
		case "supergroup", "channel":
			absID := chat.ID
			if absID < 0 {
				absID = -absID
			}
			idStr := strconv.FormatInt(absID, 10)
			if strings.HasPrefix(idStr, "100") {
				idStr = idStr[3:]
			}
			if idStr == "" {
				return nil
			}
			link := fmt.Sprintf("https://t.me/c/%s/%d", idStr, messageID)
			return &link
		default:
			return nil
		}
	}
	link := fmt.Sprintf("https://t.me/%s/%d", username, messageID)
	return &link
}
