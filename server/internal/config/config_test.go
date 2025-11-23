package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/layababa/tg_todo/server/internal/config"
)

func TestLoadConfigFallsBackToEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_VERSION", "1.2.3")
	t.Setenv("GIT_COMMIT", "abcdef123456")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	t.Setenv("REDIS_ADDR", "localhost:6380")
	t.Setenv("REDIS_NAMESPACE", "tg_todo_prod")

	cfg, err := config.Load("testdata/missing.yml")
	require.NoError(t, err)

	require.Equal(t, "production", cfg.AppEnv)
	require.Equal(t, ":9090", cfg.HTTP.Addr)
	require.Equal(t, "postgres://user:pass@localhost:5432/app?sslmode=disable", cfg.Postgres.DSN)
	require.Equal(t, "localhost:6380", cfg.Redis.Addr)
	require.Equal(t, "tg_todo_prod", cfg.Redis.Namespace)
	require.Equal(t, "1.2.3", cfg.Build.Version)
	require.Equal(t, "abcdef123456", cfg.Build.GitCommit)
}
