package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/layababa/tg_todo/server/internal/config"
	"github.com/layababa/tg_todo/server/internal/repository"
	authhandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/auth"
	"github.com/layababa/tg_todo/server/internal/server/http/handlers/healthz"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	"github.com/layababa/tg_todo/server/migrations"
	"github.com/layababa/tg_todo/server/pkg/db"
	"github.com/layababa/tg_todo/server/pkg/notion"
	pkgredis "github.com/layababa/tg_todo/server/pkg/redis"

	_ "github.com/lib/pq"
)

type dbDep struct{ db *sql.DB }

func (d dbDep) Name() string                    { return "database" }
func (d dbDep) Check(ctx context.Context) error { return d.db.PingContext(ctx) }

type redisDep struct{ rdb *redis.Client }

func (d redisDep) Name() string                    { return "redis" }
func (d redisDep) Check(ctx context.Context) error { return d.rdb.Ping(ctx).Err() }

func main() {
	// 1. Load Config
	cfg, err := config.Load("config/default.yml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Setup Logger
	var logger *zap.Logger
	if cfg.AppEnv == "production" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// 3. Connect DB
	ctx := context.Background()
	database, err := db.New(ctx, db.Config{
		DSN:             cfg.Postgres.DSN,
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
	}, "postgres")
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	if err := migrations.Run(database); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: database,
	}), &gorm.Config{})
	if err != nil {
		logger.Fatal("failed to initialize orm", zap.Error(err))
	}

	// 4. Connect Redis
	rdb, err := pkgredis.New(ctx, pkgredis.Config{
		Addr: cfg.Redis.Addr,
	})
	if err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer rdb.Close()

	// 5. Setup Gin
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(logger))

	// 6. Register Handlers
	healthHandler := healthz.NewHandler(healthz.Dependencies{
		Build: healthz.BuildInfo{
			Version:   cfg.Build.Version,
			GitCommit: cfg.Build.GitCommit,
			StartedAt: time.Now(),
		},
		Dependencies: []healthz.Dependency{
			dbDep{db: database},
			redisDep{rdb: rdb},
		},
	})
	r.GET("/healthz", healthHandler.Handle)

	userRepo := repository.NewUserRepository(gormDB)
	authHandler, err := authhandler.NewHandler(authhandler.Config{
		UserRepo: userRepo,
		NotionConfig: notion.OAuthConfig{
			ClientID:     cfg.Notion.ClientID,
			ClientSecret: cfg.Notion.ClientSecret,
			RedirectURI:  cfg.Notion.RedirectURI,
		},
		EncryptionKey: cfg.Encryption.Key,
	})
	if err != nil {
		logger.Fatal("failed to initialize auth handler", zap.Error(err))
	}

	authGroup := r.Group("/auth")
	authGroup.Use(middleware.TelegramAuth(cfg.Telegram.BotToken, userRepo))
	authGroup.GET("/status", authHandler.GetStatus)
	authGroup.GET("/notion/url", authHandler.GetNotionAuthURL)
	authGroup.POST("/notion/callback", authHandler.NotionCallback)

	// 7. Run Server
	srv := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: r,
	}

	go func() {
		logger.Info("starting server", zap.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen: %s\n", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exiting")
}
