package notion

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/internal/service/notion"
	"go.uber.org/zap"
)

type notionService interface {
	ListDatabases(ctx context.Context, userID string, query string) ([]notion.DatabaseSummary, error)
	ValidateDatabase(ctx context.Context, userID string, dbID string) (*notion.ValidationResult, error)
}

type Handler struct {
	logger  *zap.Logger
	service notionService
}

func NewHandler(logger *zap.Logger, service notionService) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
	}
}

// ListDatabases handles GET /databases
func (h *Handler) ListDatabases(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "unauthorized", "message": "Unauthorized"}})
		return
	}

	query := c.Query("search")

	dbs, err := h.service.ListDatabases(c.Request.Context(), user.ID, query)
	if err != nil {
		h.logger.Error("failed to list databases", zap.Error(err), zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "Failed to list databases"}})
		return
	}

	// Ensure empty list is [] not null
	if dbs == nil {
		dbs = []notion.DatabaseSummary{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": dbs,
		},
	})
}

// ValidateDatabase handles GET /databases/:database_id/validate
func (h *Handler) ValidateDatabase(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "unauthorized", "message": "Unauthorized"}})
		return
	}

	dbID := c.Param("database_id")
	if dbID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": "database_id is required"}})
		return
	}

	result, err := h.service.ValidateDatabase(c.Request.Context(), user.ID, dbID)
	if err != nil {
		h.logger.Error("failed to validate database", zap.Error(err), zap.String("user_id", user.ID), zap.String("db_id", dbID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "Failed to validate database"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
