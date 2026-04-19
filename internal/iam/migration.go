package iam

import (
	"context"
	"controlplane/pkg/logger"
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	mdpg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations executes any embedded SQL migrations for the IAM module.
func RunMigrations(ctx context.Context, dbURL string) error {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("migration: failed to open sql.DB: %w", err)
	}
	defer db.Close()

	driver, err := mdpg.WithInstance(db, &mdpg.Config{})
	if err != nil {
		return fmt.Errorf("migration: failed to instantiate postgres driver: %w", err)
	}

	srcDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration: failed to load embedded FS: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", srcDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migration: failed to init migrate engine: %w", err)
	}

	logger.SysInfo("iam.migration", "Running IAM database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration: failed to run up: %w", err)
	}

	logger.SysInfo("iam.migration", "IAM database migrations completed successfully")
	return nil
}
