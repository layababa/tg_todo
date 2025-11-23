package healthz

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Dependency interface {
	Name() string
	Check(ctx context.Context) error
}

type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_hash"`
	StartedAt time.Time
}

type Dependencies struct {
	Build        BuildInfo
	Dependencies []Dependency
}

type Handler struct {
	deps Dependencies
}

func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) Handle(c *gin.Context) {
	ctx := c.Request.Context()

	depStatus := make([]map[string]any, 0, len(h.deps.Dependencies))
	allUp := true

	for _, dep := range h.deps.Dependencies {
		status := "up"
		if err := dep.Check(ctx); err != nil {
			status = "down"
			allUp = false
			// We could log the error here
		}
		depStatus = append(depStatus, map[string]any{
			"name":   dep.Name(),
			"status": status,
		})
	}

	resp := gin.H{
		"success": allUp,
		"data": gin.H{
			"version":      h.deps.Build.Version,
			"git_hash":     h.deps.Build.GitCommit,
			"uptime":       time.Since(h.deps.Build.StartedAt).String(),
			"dependencies": depStatus,
		},
	}

	httpStatus := http.StatusOK
	if !allUp {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, resp)
}
