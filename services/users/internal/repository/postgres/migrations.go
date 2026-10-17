package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// RunMigrations applies the base database schema if not already initialized.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	logger.Info("running database schema migrations...")

	_, err := pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	logger.Info("database schema migrations completed successfully")
	return nil
}
