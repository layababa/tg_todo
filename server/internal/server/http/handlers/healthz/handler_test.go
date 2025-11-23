package healthz_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/zz/tg_todo/server/internal/server/http/handlers/healthz"
)

type stubDependency struct {
	name string
	err  error
}

func (s stubDependency) Name() string {
	return s.name
}

func (s stubDependency) Check(ctx context.Context) error {
	return s.err
}

func TestHandlerReturnsVersionAndGitHash(t *testing.T) {
	t.Setenv("GIN_MODE", "test")
	handler := healthz.NewHandler(healthz.Dependencies{
		Build: healthz.BuildInfo{
			Version:   "0.2.0",
			GitCommit: "deadbeef",
			StartedAt: time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		Dependencies: []healthz.Dependency{
			stubDependency{name: "database"},
			stubDependency{name: "redis"},
		},
	})

	engine := gin.New()
	engine.GET("/healthz", handler.Handle)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, true, body["success"])

	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "0.2.0", data["version"])
	require.Equal(t, "deadbeef", data["git_hash"])

	deps, ok := data["dependencies"].([]any)
	require.True(t, ok)
	require.Len(t, deps, 2)
}
