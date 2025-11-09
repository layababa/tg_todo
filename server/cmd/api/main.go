package main

import (
	"context"
	"log"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/zz/tg_todo/server/internal/config"
	"github.com/zz/tg_todo/server/internal/http"
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
		log.Fatalf("database init failed: %v", err)
	}
	defer dbConn.Close()

	if err := migrate.Apply(ctx, dbConn); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	taskRepo := task.NewRepository(dbConn)
	taskService := task.NewService(taskRepo)
	router := httpserver.NewRouter(taskService)

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	log.Printf("API server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
