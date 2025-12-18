package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
)

type Handler struct {
	logger   *zap.Logger
	userRepo repository.UserRepository
}

func NewHandler(logger *zap.Logger, userRepo repository.UserRepository) *Handler {
	return &Handler{
		logger:   logger,
		userRepo: userRepo,
	}
}

func (h *Handler) GetMe(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Simple return for now. If group stats are needed, we'd need a service or repo call.
	c.JSON(http.StatusOK, gin.H{"success": true, "data": user})
}

type UpdateSettingsRequest struct {
	Timezone          *string `json:"timezone"`
	DefaultDatabaseID *string `json:"default_database_id"`
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}
	if req.DefaultDatabaseID != nil {
		user.DefaultDatabaseID = req.DefaultDatabaseID
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		h.logger.Error("failed to update user settings", zap.Error(err), zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to update settings"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": user})
}
