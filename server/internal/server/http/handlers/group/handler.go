package group

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/service/group"
	notionsvc "github.com/layababa/tg_todo/server/internal/service/notion"
	tasksvc "github.com/layababa/tg_todo/server/internal/service/task"
	"go.uber.org/zap"
)

type Handler struct {
	logger       *zap.Logger
	groupService groupService
	taskService  *tasksvc.Service
}

type groupService interface {
	ListGroups(ctx context.Context, userID string) ([]group.GroupSummary, error)
	ValidateDatabase(ctx context.Context, userID, groupID, dbID string) (*notionsvc.ValidationResult, error)
	InitDatabase(ctx context.Context, userID, groupID, dbID string) (*notionsvc.InitResult, error)
	BindDatabase(ctx context.Context, userID, groupID, dbID string) (*models.Group, error)
	UnbindDatabase(ctx context.Context, userID, groupID string) (*models.Group, error)
}

func NewHandler(logger *zap.Logger, groupService groupService, taskService *tasksvc.Service) *Handler {
	return &Handler{
		logger:       logger,
		groupService: groupService,
		taskService:  taskService,
	}
}

func (h *Handler) ListGroups(c *gin.Context) {
	userID := c.GetString("userID")
	// role := c.Query("role") // Filter by role if needed

	groups, err := h.groupService.ListGroups(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list groups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": groups,
		},
	})
}

type ValidateRequest struct {
	DBID string `json:"db_id" binding:"required"`
}

func (h *Handler) ValidateGroupDatabase(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("group_id")

	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.groupService.ValidateDatabase(c.Request.Context(), userID, groupID, req.DBID)
	if err != nil {
		if err == group.ErrNotAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		h.logger.Error("failed to validate db", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "validation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

type InitRequest struct {
	DBID   string   `json:"db_id" binding:"required"`
	Fields []string `json:"fields"` // Optional filter
}

func (h *Handler) InitGroupDatabase(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("group_id")

	var req InitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.groupService.InitDatabase(c.Request.Context(), userID, groupID, req.DBID)
	if err != nil {
		if err == group.ErrNotAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		h.logger.Error("failed to init db", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "init failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

type BindRequest struct {
	DBID string `json:"db_id" binding:"required"`
	Mode string `json:"mode"` // replace/append
}

func (h *Handler) BindGroup(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("group_id")

	var req BindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mode handling could be added to service if needed, for MVP we just bind.
	g, err := h.groupService.BindDatabase(c.Request.Context(), userID, groupID, req.DBID)
	if err != nil {
		if err == group.ErrNotAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		if err == group.ErrGroupNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		h.logger.Error("failed to bind db", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "binding failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"group_id": g.ID,
			"db_id":    g.DatabaseID,
			"status":   g.Status,
		},
	})
}

func (h *Handler) UnbindGroup(c *gin.Context) {
	userID := c.GetString("userID")
	groupID := c.Param("group_id")

	g, err := h.groupService.UnbindDatabase(c.Request.Context(), userID, groupID)
	if err != nil {
		if err == group.ErrNotAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		if err == group.ErrGroupNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}
		h.logger.Error("failed to unbind db", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unbinding failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"group_id": g.ID,
			"status":   g.Status,
		},
	})
}

func (h *Handler) RefreshGroups(c *gin.Context) {
	// Stub
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"refreshed_at": "2023-11-20T10:00:00Z",
		},
	})
}
