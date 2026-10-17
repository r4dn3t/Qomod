package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"qomod/services/users/config"
	delivery "qomod/services/users/internal/delivery/http"
	"qomod/services/users/internal/event/rabbitmq"
	"qomod/services/users/internal/repository/postgres"
	"qomod/services/users/internal/usecase"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Root context listening for OS signals (SIGINT, SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 3. Initialize Structured Logger
	logger := setupLogger(cfg.LogLevel, cfg.AppEnv)
	logger.Info("starting users microservice",
		"env", cfg.AppEnv,
		"port", cfg.HTTPPort,
		"postgres_host", cfg.Postgres.Host,
	)

	// 4. Initialize PostgreSQL Connection Pool
	pgPool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("postgres initialization error: %w", err)
	}
	defer pgPool.Close()
	logger.Info("connected to postgres database", "db", cfg.Postgres.DBName)

	// 5. Run Database Migrations
	if cfg.RunMigrations {
		if err := postgres.RunMigrations(ctx, pgPool, logger); err != nil {
			return fmt.Errorf("database migrations failed: %w", err)
		}
	}

	// 6. Initialize RabbitMQ Publisher
	var publisher usecase.EventPublisher
	rmqPub, err := rabbitmq.NewPublisher(ctx, cfg.RabbitMQ, logger)
	if err != nil {
		logger.Warn("rabbitmq publisher initialization failed (events won't be dispatched)", "error", err)
	} else {
		defer rmqPub.Close()
		publisher = rmqPub
	}

	// 7. Initialize Repository and UseCase
	userRepo := postgres.NewUserRepository(pgPool)
	userUC := usecase.NewUserUseCase(userRepo, publisher, logger)

	// 8. Assemble HTTP Router and Server
	router := delivery.NewRouter(delivery.RouterConfig{
		Logger:  logger,
		Pool:    pgPool,
		UseCase: userUC,
	})

	server := delivery.NewServer(cfg.HTTPPort, router, logger)
	serverErrors := server.Start()

	// 9. Block until termination signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error encountered: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received, initiating graceful teardown...")
	}

	// 10. Graceful Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server graceful shutdown failed: %w", err)
	}

	logger.Info("users microservice shut down cleanly")
	return nil
}

func setupLogger(levelStr, env string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
