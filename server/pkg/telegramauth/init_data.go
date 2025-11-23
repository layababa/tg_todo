package telegramauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrExpiredData      = errors.New("init data has expired")
	ErrMissingHash      = errors.New("hash parameter is missing")
)

// InitData represents parsed Telegram Mini App init data
type InitData struct {
	QueryID      string
	User         *TelegramUser
	Receiver     *TelegramUser
	Chat         *TelegramChat
	ChatType     string
	ChatInstance string
	StartParam   string
	AuthDate     int64
	Hash         string
	RawData      string
}

// TelegramUser represents a Telegram user from init data
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
}

// TelegramChat represents a Telegram chat from init data
type TelegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
	PhotoURL string `json:"photo_url"`
}

// ParseInitData parses the raw init data string
func ParseInitData(raw string) (*InitData, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse init data: %w", err)
	}

	initData := &InitData{
		QueryID:      values.Get("query_id"),
		ChatType:     values.Get("chat_type"),
		ChatInstance: values.Get("chat_instance"),
		StartParam:   values.Get("start_param"),
		Hash:         values.Get("hash"),
		RawData:      raw,
	}

	if authDateStr := values.Get("auth_date"); authDateStr != "" {
		authDate, err := strconv.ParseInt(authDateStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid auth_date: %w", err)
		}
		initData.AuthDate = authDate
	}

	// Parse user JSON if present
	if userStr := values.Get("user"); userStr != "" {
		var user TelegramUser
		if err := json.Unmarshal([]byte(userStr), &user); err != nil {
			return nil, fmt.Errorf("failed to parse user JSON: %w", err)
		}
		initData.User = &user
	}

	return initData, nil
}

// Validate validates the init data signature using the bot token
func (d *InitData) Validate(botToken string) error {
	if d.Hash == "" {
		return ErrMissingHash
	}

	// Step 1: Create data-check-string
	dataCheckString := d.buildDataCheckString()

	// Step 2: Create secret key: HMAC-SHA256(bot_token, "WebAppData")
	secretKey := computeSecretKey(botToken)

	// Step 3: Calculate expected hash
	expectedHash := computeHash(dataCheckString, secretKey)

	// Step 4: Compare hashes
	if !hmac.Equal([]byte(expectedHash), []byte(d.Hash)) {
		return ErrInvalidSignature
	}

	return nil
}

// IsExpired checks if the init data has expired
func (d *InitData) IsExpired(maxAge time.Duration) bool {
	if d.AuthDate == 0 {
		return true
	}
	authTime := time.Unix(d.AuthDate, 0)
	return time.Since(authTime) > maxAge
}

// buildDataCheckString builds the data-check-string for validation
// Format: sorted key=value pairs (excluding 'hash'), joined by '\n'
func (d *InitData) buildDataCheckString() string {
	// Parse the raw data
	values, _ := url.ParseQuery(d.RawData)

	// Remove hash from the map
	delete(values, "hash")

	// Build sorted key=value pairs
	var pairs []string
	for key, vals := range values {
		if len(vals) > 0 {
			pairs = append(pairs, key+"="+vals[0])
		}
	}
	sort.Strings(pairs)

	return strings.Join(pairs, "\n")
}

// computeSecretKey computes the secret key from bot token
func computeSecretKey(botToken string) []byte {
	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	return h.Sum(nil)
}

// computeHash computes HMAC-SHA256 hash
func computeHash(data string, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
