package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"

	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/migrations"
	"github.com/layababa/tg_todo/server/pkg/crypto"
	"github.com/layababa/tg_todo/server/pkg/notion"
	"github.com/layababa/tg_todo/server/pkg/telegramauth"
)

const integrationTestEncryptionKey = "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE="

func TestAuthIntegration_OnboardingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	container, dsn := startPostgresContainer(t, ctx)
	defer func() {
		_ = container.Terminate(ctx)
	}()

	sqlDB := openDatabase(t, dsn)
	defer sqlDB.Close()

	require.NoError(t, migrations.Run(sqlDB))

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	require.NoError(t, err)

	repo := repository.NewUserRepository(gormDB)

	oauth := &integrationOAuthStub{
		authURL: "https://notion.example/oauth",
		tokenResp: &notion.TokenResponse{
			AccessToken:   "integration-access-token",
			WorkspaceID:   "ws_integration",
			WorkspaceName: "Integration Space",
		},
	}

	handler, err := NewHandler(Config{
		UserRepo:      repo,
		OAuthService:  oauth,
		EncryptionKey: integrationTestEncryptionKey,
	})
	require.NoError(t, err)

	logger := zap.NewNop()
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(logger))

	const botToken = "integration-bot-token"
	authGroup := router.Group("/auth")
	authGroup.Use(middleware.TelegramAuth(botToken, repo))
	authGroup.GET("/status", handler.GetStatus)
	authGroup.GET("/notion/url", handler.GetNotionAuthURL)
	authGroup.POST("/notion/callback", handler.NotionCallback)

	tgUser := &telegramauth.TelegramUser{
		ID:        68123456,
		FirstName: "Integration",
		LastName:  "Tester",
		Username:  "integration_user",
		PhotoURL:  "https://t.me/i/userpic.jpg",
	}

	initData := generateInitData(t, botToken, tgUser)

	// 1) Fetch status for a new user (should create user and report notion_connected=false)
	resp := performRequest(router, http.MethodGet, "/auth/status", nil, initData)
	require.Equal(t, http.StatusOK, resp.Code)

	var statusBody map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &statusBody))
	require.Equal(t, true, statusBody["success"])

	statusData := statusBody["data"].(map[string]any)
	require.Equal(t, false, statusData["notion_connected"])

	statusUser := statusData["user"].(map[string]any)
	require.Equal(t, "Integration Tester", statusUser["name"])

	// 2) Request Notion auth URL and capture generated state
	resp = performRequest(router, http.MethodGet, "/auth/notion/url", nil, initData)
	require.Equal(t, http.StatusOK, resp.Code)
	require.NotEmpty(t, oauth.lastState)

	// 3) Complete callback with returned state and verify persistence
	callbackPayload := fmt.Sprintf(`{"code":"valid-code","state":"%s"}`, oauth.lastState)
	resp = performRequest(router, http.MethodPost, "/auth/notion/callback", strings.NewReader(callbackPayload), initData)
	require.Equal(t, http.StatusOK, resp.Code)

	// Verify database state
	var user models.User
	require.NoError(t, gormDB.WithContext(ctx).Where("tg_id = ?", tgUser.ID).First(&user).Error)
	require.True(t, user.NotionConnected)

	var token models.UserNotionToken
	require.NoError(t, gormDB.WithContext(ctx).Where("user_id = ?", user.ID).First(&token).Error)

	accessToken, err := crypto.Decrypt(token.AccessTokenEnc, integrationTestEncryptionKey)
	require.NoError(t, err)
	require.Equal(t, oauth.tokenResp.AccessToken, accessToken)
}

// --- helpers ---

func startPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	t.Helper()

	const (
		user     = "postgres"
		password = "secret"
		dbName   = "tg_todo_auth"
	)

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		Env:          map[string]string{"POSTGRES_USER": user, "POSTGRES_PASSWORD": password, "POSTGRES_DB": dbName},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, mappedPort.Port(), dbName)
	return container, dsn
}

func openDatabase(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))

	return db
}

func performRequest(router *gin.Engine, method, path string, body io.Reader, initData string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(middleware.HeaderTelegramInitData, initData)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func generateInitData(t *testing.T, botToken string, user *telegramauth.TelegramUser) string {
	t.Helper()

	values := url.Values{}
	userJSON, err := json.Marshal(user)
	require.NoError(t, err)

	values.Set("query_id", "AAEAAAE")
	values.Set("user", string(userJSON))
	values.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))

	dataCheck := buildDataCheckString(values)
	secret := computeSecret(botToken)
	hash := computeDataHash(dataCheck, secret)

	values.Set("hash", hash)
	return values.Encode()
}

func buildDataCheckString(values url.Values) string {
	pairs := make([]string, 0, len(values))
	for key, val := range values {
		if key == "hash" || len(val) == 0 {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, val[0]))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\n")
}

func computeSecret(botToken string) []byte {
	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	return h.Sum(nil)
}

func computeDataHash(data string, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

type integrationOAuthStub struct {
	authURL   string
	tokenResp *notion.TokenResponse
	lastState string
}

func (s *integrationOAuthStub) GenerateAuthURL(state string) string {
	s.lastState = state
	return fmt.Sprintf("%s?state=%s", s.authURL, state)
}

func (s *integrationOAuthStub) ExchangeCode(ctx context.Context, code string) (*notion.TokenResponse, error) {
	if code != "valid-code" {
		return nil, fmt.Errorf("unexpected code: %s", code)
	}
	return s.tokenResp, nil
}
