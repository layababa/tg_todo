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
	"github.com/layababa/tg_todo/server/internal/models"
	"github.com/layababa/tg_todo/server/internal/repository"
	authhandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/auth"
	grouphandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/group"
	"github.com/layababa/tg_todo/server/internal/server/http/handlers/healthz"
	notionhandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/notion"
	taskhandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/task"
	telegramhandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/telegram"
	userhandler "github.com/layababa/tg_todo/server/internal/server/http/handlers/user"
	"github.com/layababa/tg_todo/server/internal/server/http/middleware"
	groupsvc "github.com/layababa/tg_todo/server/internal/service/group"
	"github.com/layababa/tg_todo/server/internal/service/notification"
	notionsvc "github.com/layababa/tg_todo/server/internal/service/notion"
	"github.com/layababa/tg_todo/server/internal/service/poller"
	"github.com/layababa/tg_todo/server/internal/service/scheduler"
	"github.com/layababa/tg_todo/server/internal/service/task"
	"github.com/layababa/tg_todo/server/internal/service/telegram"
	"github.com/layababa/tg_todo/server/migrations"
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

var defaultBotCommands = []telegram.BotCommand{
	{Command: "start", Description: "开始使用 / 打开 Mini App"},
	{Command: "help", Description: "查看帮助与功能演示"},
	{Command: "settings", Description: "打开个人设置 / 绑定 Notion"},
	{Command: "bind", Description: "群聊绑定当前 Database"},
	{Command: "todo", Description: "在群内快速创建任务"},
	{Command: "menu", Description: "显示快捷菜单"},
	{Command: "close", Description: "隐藏快捷菜单"},
}

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
	gormDB, err := gorm.Open(postgres.Open(cfg.Postgres.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	database, err := gormDB.DB()
	if err != nil {
		logger.Fatal("failed to get sql.DB", zap.Error(err))
	}
	// Configure pool
	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(25)
	database.SetConnMaxLifetime(5 * time.Minute)
	defer database.Close()

	if err := migrations.Run(database); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}

	if err := ensureLegacyConstraints(gormDB); err != nil {
		logger.Warn("failed to align legacy constraints", zap.Error(err))
	}

	// Migrate Models
	if err := gormDB.AutoMigrate(
		&models.Group{},
		&models.UserGroup{},
		&models.User{},
		&models.UserNotionToken{},
		&repository.Task{},
		&repository.TaskAssignee{},
		&repository.TaskContextSnapshot{},
		&repository.TaskEvent{},
		&repository.TaskComment{},
	); err != nil {
		logger.Fatal("failed to migrate models", zap.Error(err))
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
	r.Use(gin.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.CORS())

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

	// API Route Group
	api := r.Group("/api")

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

	authGroup := api.Group("/auth")
	authGroup.Use(middleware.TelegramAuth(cfg.Telegram.BotToken, userRepo))
	authGroup.GET("/status", authHandler.GetStatus)
	authGroup.GET("/notion/url", authHandler.GetNotionAuthURL)
	authGroup.POST("/notion/callback", authHandler.NotionCallback)

	// Notion Service
	notionService := notionsvc.NewService(logger, userRepo, cfg.Encryption.Key)
	notionHandler := notionhandler.NewHandler(logger, notionService)

	dbGroup := api.Group("/databases")
	dbGroup.Use(middleware.TelegramAuth(cfg.Telegram.BotToken, userRepo))
	dbGroup.GET("", notionHandler.ListDatabases)
	dbGroup.GET("/:database_id/validate", notionHandler.ValidateDatabase)

	// Groups Service
	groupRepo := repository.NewGroupRepository(gormDB)
	userGroupRepo := repository.NewUserGroupRepository(gormDB)
	groupService := groupsvc.NewService(logger, groupRepo, notionService)

	// Telegram Client (Hoist for Notification Service)
	tgClient := telegram.NewClient(cfg.Telegram.BotToken)
	if err := tgClient.SetMyCommands(telegram.SetMyCommandsRequest{Commands: defaultBotCommands}); err != nil {
		logger.Warn("failed to set default telegram bot commands", zap.Error(err))
	}
	if err := tgClient.SetMyCommands(telegram.SetMyCommandsRequest{
		Commands: defaultBotCommands,
		Scope:    &telegram.CommandScope{Type: telegram.CommandScopeAllPrivateChats},
	}); err != nil {
		logger.Warn("failed to set private telegram bot commands", zap.Error(err))
	}
	if err := tgClient.SetMyCommands(telegram.SetMyCommandsRequest{
		Commands: defaultBotCommands,
		Scope:    &telegram.CommandScope{Type: telegram.CommandScopeAllGroupChats},
	}); err != nil {
		logger.Warn("failed to set group telegram bot commands", zap.Error(err))
	}

	taskRepo := repository.NewTaskRepository(gormDB)
	logger.Info("Initializing notification service",
		zap.String("bot_name", cfg.Telegram.BotName),
		zap.String("app_short_name", cfg.Telegram.AppShortName))
	notificationService := notification.NewService(logger, taskRepo, userRepo, tgClient, cfg.Telegram.BotName, cfg.Telegram.AppShortName)

	// -- Task Service (Injects Notification Service)
	taskService := task.NewService(task.ServiceConfig{
		Logger:        logger,
		Repo:          taskRepo,
		UserRepo:      userRepo,
		Notifier:      notificationService,
		EncryptionKey: cfg.Encryption.Key,
	})

	// -- Scheduler Service (Daily Digest)
	schedulerService := scheduler.NewService(logger, userRepo, taskRepo, notificationService, tgClient)
	schedulerService.Start()
	// defer schedulerService.Stop() // Optional: Stop on graceful shutdown

	// -- Notion Poller Service
	pollerService := poller.NewPoller(groupRepo, taskService, notionService, cfg.Encryption.Key)
	pollerService.Start(ctx)
	defer pollerService.Stop()

	// -- Handlers
	taskHandler := taskhandler.NewHandler(logger, taskService, userGroupRepo)
	userHandler := userhandler.NewHandler(logger, userRepo)

	// Group Handler needs Task Service now
	groupHandler := grouphandler.NewHandler(logger, groupService, taskService)

	groupGroup := api.Group("/groups")
	groupGroup.Use(middleware.TelegramAuth(cfg.Telegram.BotToken, userRepo))
	groupGroup.GET("", groupHandler.ListGroups)
	groupGroup.POST("/refresh", groupHandler.RefreshGroups)
	groupGroup.POST("/:group_id/bind", groupHandler.BindGroup)
	groupGroup.POST("/:group_id/unbind", groupHandler.UnbindGroup)
	groupGroup.POST("/:group_id/db/validate", groupHandler.ValidateGroupDatabase)
	groupGroup.POST("/:group_id/db/init", groupHandler.InitGroupDatabase)

	taskGroup := api.Group("/tasks")
	taskGroup.Use(middleware.TelegramAuth(cfg.Telegram.BotToken, userRepo))
	taskGroup.GET("", taskHandler.List)
	taskGroup.GET("/counts", taskHandler.GetCounts)
	taskGroup.GET("/:task_id", taskHandler.Get)
	taskGroup.PATCH("/:task_id", taskHandler.Update)
	taskGroup.DELETE("/:task_id", taskHandler.Delete)
	taskGroup.POST("", taskHandler.CreateWebTask)
	taskGroup.GET("/:task_id/comments", taskHandler.ListComments)
	taskGroup.POST("/:task_id/comments", taskHandler.CreateComment)

	meGroup := api.Group("/me")
	meGroup.Use(middleware.TelegramAuth(cfg.Telegram.BotToken, userRepo))
	meGroup.GET("", userHandler.GetMe)
	meGroup.PATCH("/settings", userHandler.UpdateSettings)

	// Telegram Webhook (Keeping at root or moving to /api/webhook)
	// Let's keep it at /webhook/telegram for now as it's typically configured once in BotFather
	tgUpdateRepo := repository.NewTelegramUpdateRepository(gormDB)
	taskCreator := task.NewCreator(task.CreatorConfig{
		Logger:      logger,
		TaskRepo:    taskRepo,
		TaskService: taskService,
		UpdateRepo:  tgUpdateRepo,
		UserRepo:    userRepo,
		GroupRepo:   groupRepo,
	})
	deduplicator := telegram.NewDeduplicator(rdb)

	tgHandler := telegramhandler.NewHandler(telegramhandler.Config{
		Logger:       logger,
		Deduplicator: deduplicator,
		Repo:         tgUpdateRepo,
		UserRepo:     userRepo,
		TaskCreator:  taskCreator,
		TaskService:  taskService, // Injected TaskService
		GroupService: groupService,
		TgClient:     tgClient,
		SecretToken:  os.Getenv("TELEGRAM_SECRET_TOKEN"),
		BotUsername:  cfg.Telegram.BotName,
		WebAppURL:    cfg.Telegram.WebAppURL,
	})
	r.POST("/webhook/telegram", tgHandler.HandleWebhook)

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

func ensureLegacyConstraints(db *gorm.DB) error {
	stmt := `
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM information_schema.tables WHERE table_name = 'users'
	) THEN
		RETURN;
	END IF;

	IF EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'users_tg_id_key'
	) AND NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'uni_users_tg_id'
	) THEN
		ALTER TABLE users RENAME CONSTRAINT users_tg_id_key TO uni_users_tg_id;
	END IF;
END
$$;
`
	return db.Exec(stmt).Error
}
