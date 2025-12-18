package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/pkg/telegramauth"
)

const (
	HeaderTelegramInitData = "X-Telegram-Init-Data"
	ContextKeyUser         = "user"
	MaxInitDataAge         = 24 * time.Hour // Init data expires after 24 hours
)

// TelegramAuth creates a middleware that validates Telegram init data
// and loads or creates the user
func TelegramAuth(botToken string, userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get init data from header
		initDataRaw := c.GetHeader(HeaderTelegramInitData)
		if initDataRaw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "missing_init_data",
					"message": "Telegram init data is required",
				},
			})
			return
		}

		// Parse init data
		initData, err := telegramauth.ParseInitData(initDataRaw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_init_data",
					"message": "Failed to parse init data",
				},
			})
			return
		}

		// Validate signature
		if err := initData.Validate(botToken); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_signature",
					"message": "Init data signature is invalid",
				},
			})
			return
		}

		// Check expiration
		if initData.IsExpired(MaxInitDataAge) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "expired_init_data",
					"message": "Init data has expired",
				},
			})
			return
		}

		// Get user info from init data
		if initData.User == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "missing_user_data",
					"message": "User data not found in init data",
				},
			})
			return
		}

		// Try to find existing user
		user, err := userRepo.FindByTgID(c.Request.Context(), initData.User.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "database_error",
					"message": "Failed to lookup user",
				},
			})
			return
		}

		// Create new user if not found
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = &models.User{
				TgID:       initData.User.ID,
				TgUsername: initData.User.Username,
				Name:       buildUserName(initData.User),
				PhotoURL:   initData.User.PhotoURL,
				Timezone:   "UTC+0", // Default timezone
			}
			if err := userRepo.Create(c.Request.Context(), user); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "user_creation_failed",
						"message": "Failed to create user",
					},
				})
				return
			}
		} else {
			// Sync user profile from Telegram (name, photo, username may change)
			needsUpdate := false
			newName := buildUserName(initData.User)
			if user.Name != newName && newName != "User" {
				user.Name = newName
				needsUpdate = true
			}
			if user.TgUsername != initData.User.Username {
				user.TgUsername = initData.User.Username
				needsUpdate = true
			}
			if user.PhotoURL != initData.User.PhotoURL && initData.User.PhotoURL != "" {
				user.PhotoURL = initData.User.PhotoURL
				needsUpdate = true
			}
			if needsUpdate {
				_ = userRepo.Update(c.Request.Context(), user) // Best effort, don't block auth
			}
		}

		// Store user in context
		c.Set(ContextKeyUser, user)

		// Continue to next handler
		c.Next()
	}
}

// buildUserName constructs a display name from Telegram user data
func buildUserName(tgUser *telegramauth.TelegramUser) string {
	if tgUser.FirstName != "" && tgUser.LastName != "" {
		return tgUser.FirstName + " " + tgUser.LastName
	}
	if tgUser.FirstName != "" {
		return tgUser.FirstName
	}
	if tgUser.Username != "" {
		return "@" + tgUser.Username
	}
	return "User"
}

// GetUserFromContext retrieves the authenticated user from the Gin context
func GetUserFromContext(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil, false
	}
	u, ok := user.(*models.User)
	return u, ok
}
