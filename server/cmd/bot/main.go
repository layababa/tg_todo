package main

import (
	"context"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/zz/tg_todo/server/internal/bot"
	"github.com/zz/tg_todo/server/internal/config"
	"github.com/zz/tg_todo/server/internal/migrate"
	"github.com/zz/tg_todo/server/internal/task"
	"github.com/zz/tg_todo/server/pkg/db"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	dbConn, err := db.New(ctx, db.Config{
		DSN:             cfg.PostgresURL,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	}, "pgx")
	if err != nil {
		log.Fatalf("bot: database init failed: %v", err)
	}
	defer dbConn.Close()

	if err := migrate.Apply(ctx, dbConn); err != nil {
		log.Fatalf("bot: migrations failed: %v", err)
	}

	taskRepo := task.NewRepository(dbConn)
	taskService := task.NewService(taskRepo)

	tgBot := bot.New(cfg.TelegramToken, cfg.TelegramAPIBase, taskService)
	if err := tgBot.Start(ctx); err != nil {
		log.Fatalf("bot exited: %v", err)
	}
}
