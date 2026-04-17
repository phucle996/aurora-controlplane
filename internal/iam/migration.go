package iam

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"controlplane/pkg/logger"

	"github.com/golang-migrate/migrate/v4"
	mdpg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations executes any embedded SQL migrations for the IAM module.
func RunMigrations(ctx context.Context, dbURL string) error {
	// 1. Establish native database/sql connection for migration driver
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("migration: failed to open sql.DB: %w", err)
	}
	defer db.Close()

	// 2. Wrap using postgres driver
	driver, err := mdpg.WithInstance(db, &mdpg.Config{})
	if err != nil {
		return fmt.Errorf("migration: failed to instantiate postgres driver: %w", err)
	}

	// 3. Mount embedded directory
	srcDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration: failed to load embedded FS: %w", err)
	}

	// 4. Initialize Migrate
	m, err := migrate.NewWithInstance("iofs", srcDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migration: failed to init migrate engine: %w", err)
	}

	// 5. Run Up
	logger.SysInfo("iam.migration", "starting", "Running IAM database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration: failed to run up: %w", err)
	}

	logger.SysInfo("iam.migration", "complete", "IAM database migrations completed successfully")
	return nil
}
