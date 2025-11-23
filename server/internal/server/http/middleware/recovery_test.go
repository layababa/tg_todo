package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/zz/tg_todo/server/internal/server/http/middleware"
)

func TestRecoveryMiddlewareReturnsJSONError(t *testing.T) {
	t.Setenv("GIN_MODE", "test")
	logger := zaptest.NewLogger(t)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Recovery(logger))

	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set(middleware.HeaderRequestID, "req-123")
	resp := httptest.NewRecorder()

	engine.ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))

	require.Equal(t, false, body["success"])
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "internal_server_error", errObj["code"])

	meta, ok := body["meta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "req-123", meta["request_id"])
}
