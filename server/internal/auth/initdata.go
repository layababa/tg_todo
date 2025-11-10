package auth

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

// ErrInvalidInitData 表示 initData 缺失或签名校验失败。
var ErrInvalidInitData = errors.New("invalid telegram init data")

// Validator 负责校验 Telegram WebApp 传入的 initData。
type Validator struct {
	secret []byte
}

// NewValidator 按照 Telegram 文档使用 Bot Token 推导校验密钥。
func NewValidator(botToken string) (*Validator, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, fmt.Errorf("bot token is empty")
	}
	sum := sha256.Sum256([]byte("WebAppData" + botToken))
	secret := sum[:]
	return &Validator{secret: secret}, nil
}

// InitData 解析后的 initData 结构。
type InitData struct {
	Hash     string
	Raw      string
	User     TelegramUser
	QueryID  string
	AuthDate time.Time
	Data     map[string]string
}

// TelegramUser 对应 initData 中的 user 字段。
type TelegramUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
}

// Validate 校验签名并返回结构化的 initData。
func (v *Validator) Validate(rawInitData string) (InitData, error) {
	if strings.TrimSpace(rawInitData) == "" {
		return InitData{}, fmt.Errorf("%w: empty data", ErrInvalidInitData)
	}
	values, err := url.ParseQuery(rawInitData)
	if err != nil {
		return InitData{}, fmt.Errorf("%w: parse query: %v", ErrInvalidInitData, err)
	}
	hash := values.Get("hash")
	if hash == "" {
		return InitData{}, fmt.Errorf("%w: hash missing", ErrInvalidInitData)
	}
	data := make(map[string]string, len(values))
	var dataCheck []string
	for key, val := range values {
		if key == "hash" {
			continue
		}
		value := ""
		if len(val) > 0 {
			value = val[0]
			data[key] = value
		}
		dataCheck = append(dataCheck, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(dataCheck)
	dataCheckString := strings.Join(dataCheck, "\n")

	mac := hmac.New(sha256.New, v.secret)
	if _, err := mac.Write([]byte(dataCheckString)); err != nil {
		return InitData{}, fmt.Errorf("build hmac: %w", err)
	}
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(hash))) {
		return InitData{}, ErrInvalidInitData
	}

	var tgUser TelegramUser
	if userStr, ok := data["user"]; ok && userStr != "" {
		if err := json.Unmarshal([]byte(userStr), &tgUser); err != nil {
			return InitData{}, fmt.Errorf("%w: decode user: %v", ErrInvalidInitData, err)
		}
	}

	var authDate time.Time
	if authTimestamp := data["auth_date"]; authTimestamp != "" {
		if ts, err := parseAuthTimestamp(authTimestamp); err == nil {
			authDate = ts
		}
	}

	return InitData{
		Hash:     hash,
		Raw:      rawInitData,
		User:     tgUser,
		QueryID:  data["query_id"],
		AuthDate: authDate,
		Data:     data,
	}, nil
}

func parseAuthTimestamp(val string) (time.Time, error) {
	seconds, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(seconds, 0), nil
}
