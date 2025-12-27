package task

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/internal/service/task"
	"go.uber.org/zap"
)

type taskService interface {
	ListTasks(ctx context.Context, userID string, params task.ListParams) ([]task.TaskDetail, error)
	GetTask(ctx context.Context, id string) (*repository.Task, error)
	CreateWebTask(ctx context.Context, userID, title, description string) (*repository.Task, error)
	UpdateTask(ctx context.Context, id string, params task.UpdateParams) (*repository.Task, error)
	DeleteTask(ctx context.Context, id string) error

	// Comment methods
	CreateComment(ctx context.Context, taskID, userID, content string, parentID *string) (*repository.TaskComment, error)
	ListComments(ctx context.Context, taskID string) ([]repository.TaskComment, error)
	GetTaskCounts(ctx context.Context, userID string) (*repository.TaskCounts, error)
}

type Handler struct {
	logger        *zap.Logger
	service       taskService
	userGroupRepo repository.UserGroupRepository
}

func NewHandler(logger *zap.Logger, service taskService, userGroupRepo repository.UserGroupRepository) *Handler {
	return &Handler{
		logger:        logger,
		service:       service,
		userGroupRepo: userGroupRepo,
	}
}

func (h *Handler) List(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	view := repository.TaskView(c.DefaultQuery("view", string(repository.TaskViewAll)))
	switch view {
	case repository.TaskViewAll, repository.TaskViewAssigned, repository.TaskViewCreated, repository.TaskViewDone:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "invalid_view", "message": "invalid view"}})
		return
	}

	var dbID *string
	if v := c.Query("database_id"); v != "" {
		dbID = &v
	}

	limit := parseIntWithDefault(c.Query("limit"), 20)
	offset := parseIntWithDefault(c.Query("offset"), 0)

	items, err := h.service.ListTasks(c.Request.Context(), user.ID, task.ListParams{
		View:       view,
		DatabaseID: dbID,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		h.logger.Error("list tasks failed", zap.Error(err), zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to list tasks"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items}})
}

func (h *Handler) GetCounts(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	counts, err := h.service.GetTaskCounts(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error("get task counts failed", zap.Error(err), zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to get task counts"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": counts})
}

func (h *Handler) Get(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	_ = user // placeholder for future ownership check

	id := c.Param("task_id")
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "task not found"}})
			return
		}
		h.logger.Error("get task failed", zap.Error(err), zap.String("task_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to get task"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": task})
}

func (h *Handler) Delete(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	id := c.Param("task_id")

	// Get existing task to check permission
	existingTask, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "task not found"}})
			return
		}
		h.logger.Error("get task failed", zap.Error(err), zap.String("task_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to get task"}})
		return
	}

	// Check permission
	canModify, err := task.CanModifyTask(c.Request.Context(), user.ID, existingTask, h.userGroupRepo)
	if err != nil {
		h.logger.Error("permission check failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "permission check failed"}})
		return
	}

	if !canModify {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "forbidden",
				"message": "您没有权限删除此任务。只有创建人、指派人或群管理员可以删除任务。",
			},
		})
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "task not found"}})
			return
		}
		h.logger.Error("delete task failed", zap.Error(err), zap.String("task_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to delete task"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type UpdateRequest struct {
	Title  *string                `json:"title"`
	Status *repository.TaskStatus `json:"status"`
	DueAt  *time.Time             `json:"due_at"`
}

func (h *Handler) Update(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	id := c.Param("task_id")

	// Get existing task to check permission
	existingTask, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "task not found"}})
			return
		}
		h.logger.Error("get task failed", zap.Error(err), zap.String("task_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to get task"}})
		return
	}

	// Check permission
	canModify, err := task.CanModifyTask(c.Request.Context(), user.ID, existingTask, h.userGroupRepo)
	if err != nil {
		h.logger.Error("permission check failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "permission check failed"}})
		return
	}

	if !canModify {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "forbidden",
				"message": "您没有权限修改此任务。只有创建人、指派人或群管理员可以修改任务。",
			},
		})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	updatedTask, err := h.service.UpdateTask(c.Request.Context(), id, task.UpdateParams{
		Title:  req.Title,
		Status: req.Status,
		DueAt:  req.DueAt,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "task not found"}})
			return
		}
		h.logger.Error("update task failed", zap.Error(err), zap.String("task_id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to update task"}})
		return
	}

	// Helper to handle wrapping if needed, but currently Get returns flat Task.
	// We return flat Task here too.
	c.JSON(http.StatusOK, gin.H{"success": true, "data": updatedTask})
}

type CreateTaskRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) CreateWebTask(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	task, err := h.service.CreateWebTask(c.Request.Context(), user.ID, req.Title, req.Description)
	if err != nil {
		h.logger.Error("create task failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to create task"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": task})
}

type CreateCommentRequest struct {
	Content  string  `json:"content" binding:"required"`
	ParentID *string `json:"parent_id"`
}

func (h *Handler) ListComments(c *gin.Context) {
	taskID := c.Param("task_id")
	comments, err := h.service.ListComments(c.Request.Context(), taskID)
	if err != nil {
		h.logger.Error("list comments failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to list comments"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": comments})
}

func (h *Handler) CreateComment(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	taskID := c.Param("task_id")
	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "invalid_request", "message": err.Error()}})
		return
	}

	comment, err := h.service.CreateComment(c.Request.Context(), taskID, user.ID, req.Content, req.ParentID)
	if err != nil {
		h.logger.Error("create comment failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": "failed to create comment"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": comment})
}

func parseIntWithDefault(val string, def int) int {
	if val == "" {
		return def
	}
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	return def
}
