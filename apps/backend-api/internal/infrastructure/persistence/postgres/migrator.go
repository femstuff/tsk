package postgres

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// ApplySchema is kept for backward compatibility in tests; prefer RunMigrations.
func ApplySchema(ctx context.Context, pool *pgxpool.Pool) error {
	return RunMigrations(ctx, pool)
}
