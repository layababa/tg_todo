package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/pkg/crypto"
	"github.com/layababa/tg_todo/server/pkg/notion"
)

// Handler wires auth related HTTP endpoints.
type Handler struct {
	userRepo      repository.UserRepository
	oauth         oauthService
	encryptionKey string
	stateCodec    stateCodec
}

// Config holds the dependencies required by Handler.
type Config struct {
	UserRepo      repository.UserRepository
	NotionConfig  notion.OAuthConfig
	OAuthService  oauthService
	EncryptionKey string
}

// NewHandler builds a Handler instance with sane defaults.
func NewHandler(cfg Config) (*Handler, error) {
	if cfg.UserRepo == nil {
		return nil, errors.New("user repository is required")
	}
	if cfg.EncryptionKey == "" {
		return nil, errors.New("encryption key is required")
	}

	stateCodec, err := newHMACStateCodec(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}

	client := cfg.OAuthService
	if client == nil {
		client = &notionOAuthClient{config: cfg.NotionConfig}
	}

	return &Handler{
		userRepo:      cfg.UserRepo,
		oauth:         client,
		encryptionKey: cfg.EncryptionKey,
		stateCodec:    stateCodec,
	}, nil
}

// GetStatus returns the authenticated user profile and onboarding hints.
func (h *Handler) GetStatus(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "unauthorized",
				"message": "user not found in context",
			},
		})
		return
	}

	redirectHint := c.Query("start_param")
	if redirectHint == "" {
		// Fallback to start_param from signed init_data
		if initData, ok := middleware.GetInitDataFromContext(c); ok {
			redirectHint = initData.StartParam
		}
	}

	var hint any
	if redirectHint != "" {
		hint = redirectHint
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": gin.H{
				"id":                user.ID,
				"tg_id":             user.TgID,
				"name":              user.Name,
				"photo_url":         user.PhotoURL,
				"timezone":          user.Timezone,
				"notion_connected":  user.NotionConnected,
				"username":          user.TgUsername,
				"telegram_photo":    user.PhotoURL,
				"telegram_username": user.TgUsername,
			},
			"notion_connected":   user.NotionConnected,
			"pending_sync_count": 0,
			"redirect_hint":      hint,
		},
	})
}

// GetNotionAuthURL generates the Notion OAuth URL with a signed state.
func (h *Handler) GetNotionAuthURL(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "unauthorized",
				"message": "user not found in context",
			},
		})
		return
	}

	state, err := h.stateCodec.Encode(user.TgID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "state_generation_failed",
				"message": "failed to generate oauth state",
			},
		})
		return
	}

	authURL := h.oauth.GenerateAuthURL(state)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": authURL,
		},
	})
}

type notionCallbackRequest struct {
	Code  string `form:"code" json:"code" binding:"required"`
	State string `form:"state" json:"state" binding:"required"`
}

// NotionCallback exchanges authorization code and persists encrypted token.
func (h *Handler) NotionCallback(c *gin.Context) {
	var req notionCallbackRequest
	if err := c.ShouldBind(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_request",
				"message": "code and state are required",
			},
		})
		return
	}

	tgID, err := h.stateCodec.Decode(req.State)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "invalid_state",
				"message": "failed to verify oauth state",
			},
		})
		return
	}

	ctx := c.Request.Context()
	user, err := h.userRepo.FindByTgID(ctx, tgID)
	if err != nil {
		status := http.StatusInternalServerError
		errPayload := gin.H{
			"code":    "user_lookup_failed",
			"message": "failed to load user",
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			errPayload = gin.H{
				"code":    "user_not_found",
				"message": "user not found for provided state",
			}
		}
		c.AbortWithStatusJSON(status, gin.H{
			"success": false,
			"error":   errPayload,
		})
		return
	}

	tokenResp, err := h.oauth.ExchangeCode(ctx, req.Code)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "notion_exchange_failed",
				"message": "failed to exchange auth code with notion",
			},
		})
		return
	}

	accessTokenEnc, err := crypto.Encrypt(tokenResp.AccessToken, h.encryptionKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "token_encryption_failed",
				"message": "failed to encrypt notion token",
			},
		})
		return
	}

	refreshTokenEnc, err := crypto.Encrypt("", h.encryptionKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "token_encryption_failed",
				"message": "failed to encrypt notion token",
			},
		})
		return
	}

	token := &models.UserNotionToken{
		UserID:          user.ID,
		AccessTokenEnc:  accessTokenEnc,
		RefreshTokenEnc: refreshTokenEnc,
		WorkspaceID:     tokenResp.WorkspaceID,
		WorkspaceName:   tokenResp.WorkspaceName,
	}

	if err := h.userRepo.SaveNotionToken(ctx, token); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "token_persist_failed",
				"message": "failed to save notion token",
			},
		})
		return
	}

	user.NotionConnected = true
	if err := h.userRepo.Update(ctx, user); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "user_update_failed",
				"message": "failed to update user",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"notion_connected": true,
		},
	})
}

type oauthService interface {
	GenerateAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*notion.TokenResponse, error)
}

type notionOAuthClient struct {
	config notion.OAuthConfig
}

func (n *notionOAuthClient) GenerateAuthURL(state string) string {
	return notion.GenerateAuthURL(n.config, state)
}

func (n *notionOAuthClient) ExchangeCode(ctx context.Context, code string) (*notion.TokenResponse, error) {
	return notion.ExchangeCode(ctx, n.config, code)
}

type stateCodec interface {
	Encode(tgID int64) (string, error)
	Decode(state string) (int64, error)
}

type hmacStateCodec struct {
	key []byte
}

func newHMACStateCodec(base64Key string) (*hmacStateCodec, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key encoding: %w", err)
	}
	if len(key) == 0 {
		return nil, errors.New("encryption key cannot be empty")
	}
	return &hmacStateCodec{key: key}, nil
}

func (h *hmacStateCodec) Encode(tgID int64) (string, error) {
	payload := strconv.FormatInt(tgID, 10)
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	raw := payload + ":" + signature
	return base64.RawURLEncoding.EncodeToString([]byte(raw)), nil
}

func (h *hmacStateCodec) Decode(state string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		return 0, fmt.Errorf("decode state: %w", err)
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return 0, errors.New("invalid state format")
	}

	payload := parts[0]
	signature := parts[1]

	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return 0, errors.New("invalid state signature")
	}

	tgID, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid state payload: %w", err)
	}

	return tgID, nil
}
