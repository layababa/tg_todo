package calendar

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
)

// Handler handles calendar-related requests
type Handler struct {
	logger   *zap.Logger
	userRepo repository.UserRepository
	taskRepo repository.TaskRepository
	baseURL  string // e.g., "https://api.yoursite.com"
}

// Config holds configuration for the handler
type Config struct {
	Logger   *zap.Logger
	UserRepo repository.UserRepository
	TaskRepo repository.TaskRepository
	BaseURL  string
}

// NewHandler creates a new calendar handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		logger:   cfg.Logger,
		userRepo: cfg.UserRepo,
		taskRepo: cfg.TaskRepo,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
	}
}

// GenerateCalendarToken generates or rotates the calendar token for the current user
func (h *Handler) GenerateCalendarToken(c *gin.Context) {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Generate a new UUID token
	newToken := uuid.New().String()
	user.CalendarToken = &newToken

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		h.logger.Error("failed to update calendar token", zap.Error(err), zap.String("user_id", user.ID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "internal_error", "message": "failed to generate calendar token"},
		})
		return
	}

	// Build the webcal URL
	webcalURL := fmt.Sprintf("webcal://%s/cal/%s/todo.ics", strings.TrimPrefix(h.baseURL, "https://"), newToken)
	httpsURL := fmt.Sprintf("%s/cal/%s/todo.ics", h.baseURL, newToken)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token":      newToken,
			"webcal_url": webcalURL,
			"https_url":  httpsURL,
		},
	})
}

// GetCalendarFeed returns the ICS calendar feed for a user
func (h *Handler) GetCalendarFeed(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	ctx := c.Request.Context()

	// Find user by calendar token
	user, err := h.userRepo.FindByCalendarToken(ctx, token)
	if err != nil {
		h.logger.Warn("calendar token not found", zap.String("token", token))
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Fetch incomplete tasks with due dates for this user
	filter := repository.TaskListFilter{
		View: repository.TaskViewAssigned,
	}
	tasks, err := h.taskRepo.ListByUser(ctx, user.ID, filter)
	if err != nil {
		h.logger.Error("failed to fetch tasks for calendar", zap.Error(err), zap.String("user_id", user.ID))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Generate ICS content
	icsContent := h.generateICS(tasks, user.Name)

	// Set headers for ICS
	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"todo.ics\"")
	c.String(http.StatusOK, icsContent)
}

// generateICS creates the ICS file content from tasks
func (h *Handler) generateICS(tasks []repository.Task, calendarName string) string {
	var sb strings.Builder

	// ICS header
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//TG Todo//Telegram To-Do Mini App//EN\r\n")
	sb.WriteString(fmt.Sprintf("X-WR-CALNAME:%s 的待办\r\n", escapeICS(calendarName)))
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	now := time.Now().UTC()
	dtstamp := now.Format("20060102T150405Z")

	for _, task := range tasks {
		// Only include tasks with due dates
		if task.DueAt == nil {
			continue
		}

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s@tgtodo\r\n", task.ID))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", dtstamp))

		// All-day event: use DATE format (YYYYMMDD)
		dueDate := task.DueAt.Format("20060102")
		sb.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", dueDate))
		sb.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", dueDate))

		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(task.Title)))

		// Description: Include status
		description := fmt.Sprintf("Status: %s", task.Status)
		if task.Description != "" {
			description += "\n" + task.Description
		}
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(description)))

		// Reminder 1 hour before (if due time is set)
		sb.WriteString("BEGIN:VALARM\r\n")
		sb.WriteString("ACTION:DISPLAY\r\n")
		sb.WriteString("TRIGGER:-PT1H\r\n")
		sb.WriteString(fmt.Sprintf("DESCRIPTION:任务提醒: %s\r\n", escapeICS(task.Title)))
		sb.WriteString("END:VALARM\r\n")

		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

// escapeICS escapes special characters for ICS format
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
