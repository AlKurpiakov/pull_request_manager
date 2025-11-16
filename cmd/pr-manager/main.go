package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"prmanager/internal/api"
	"prmanager/internal/config"
	"prmanager/internal/migration"
	"prmanager/internal/repository/postgres"
	"prmanager/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.LoadFromEnv()
	logger.Info("application starting", "config", map[string]interface{}{
		"port":    cfg.Port,
		"db_host": os.Getenv("DB_HOST"),
	})

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBConn)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migration.Run(ctx, pool); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	repo := postgres.NewRepo(pool)
	svc := service.NewService(repo, logger)
	h := api.NewHandler(svc, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Router(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	logger.Info("server starting", "port", cfg.Port, "address", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
