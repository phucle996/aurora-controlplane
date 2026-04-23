package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"controlplane/internal/app/bootstrap"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHotpathIndexesExistAfterMigrations(t *testing.T) {
	db := mustOpenIAMIntegrationDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'iam'
		  AND indexname = ANY($1)
	`, []string{
		"idx_iam_refresh_tokens_user_id",
		"idx_iam_refresh_tokens_expires_at",
		"idx_iam_devices_user_id_fingerprint_last_active_at",
	})
	if err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan index name: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate indexes: %v", err)
	}

	for _, name := range []string{
		"idx_iam_refresh_tokens_user_id",
		"idx_iam_refresh_tokens_expires_at",
		"idx_iam_devices_user_id_fingerprint_last_active_at",
	} {
		if !found[name] {
			t.Fatalf("expected index %q to exist after migrations", name)
		}
	}
}

func mustOpenIAMIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("IAM_IT_DB_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("set IAM_IT_DB_URL (or TEST_DATABASE_URL) to run IAM migration integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open db pool: %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	if err := bootstrap.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return db
}
