package bootstrap

import (
	"context"
	"controlplane/internal/config"
	"controlplane/pkg/logger"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	mdpg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func RunMigrations(ctx context.Context, cfg *config.Config) error {
	migrationDir, err := resolveIAMMigrationDir()
	if err != nil {
		return err
	}

	db, err := sql.Open("postgres", buildPostgresDSN(&cfg.Psql))
	if err != nil {
		return fmt.Errorf("migration: failed to open sql.DB: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("migration: failed to ping postgres: %w", err)
	}

	driver, err := mdpg.WithInstance(db, &mdpg.Config{})
	if err != nil {
		return fmt.Errorf("migration: failed to instantiate postgres driver: %w", err)
	}

	sourceURL := "file://" + filepath.ToSlash(migrationDir)
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migration: failed to init migrate engine: %w", err)
	}

	logger.SysInfo("app.migration", fmt.Sprintf("Running IAM database migrations from %s", migrationDir))
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration: failed to run up: %w", err)
	}

	logger.SysInfo("app.migration", "IAM database migrations completed successfully")
	return nil
}

func buildPostgresDSN(cfg *config.PsqlCfg) string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	if !cfg.TLSEnabled {
		return dsn
	}

	sslMode := strings.TrimSpace(cfg.SSLMode)
	if sslMode == "" || strings.EqualFold(sslMode, "disable") {
		sslMode = "verify-full"
	}

	dsn = fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode,
	)

	if cfg.CACertPath != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", cfg.CACertPath)
	}
	if cfg.CertPath != "" {
		dsn += fmt.Sprintf(" sslcert=%s", cfg.CertPath)
	}
	if cfg.KeyPath != "" {
		dsn += fmt.Sprintf(" sslkey=%s", cfg.KeyPath)
	}

	return dsn
}

func resolveIAMMigrationDir() (string, error) {
	candidates := []string{
		"/etc/aurora-controlplane/migrations/iam",
		filepath.Join("internal", "iam", "migrations"),
		filepath.Join("aurora-controlplane", "internal", "iam", "migrations"),
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "internal", "iam", "migrations"),
			filepath.Join(wd, "aurora-controlplane", "internal", "iam", "migrations"),
		)
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("migration: iam migration directory not found")
}
