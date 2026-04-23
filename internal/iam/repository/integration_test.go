package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"controlplane/internal/app/bootstrap"

	"github.com/jackc/pgx/v5/pgxpool"
)

func mustOpenIAMRepositoryIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("IAM_IT_DB_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("set IAM_IT_DB_URL (or TEST_DATABASE_URL) to run IAM repository integration tests")
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

func mustExecIAM(t *testing.T, db *pgxpool.Pool, query string, args ...any) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.Exec(ctx, query, args...); err != nil {
		t.Fatalf("exec query failed: %v\nquery: %s", err, query)
	}
}

func mustQueryRowIAM(t *testing.T, db *pgxpool.Pool, query string, args ...any) pgxRow {
	t.Helper()
	return pgxRow{t: t, db: db, query: query, args: args}
}

type pgxRow struct {
	t     *testing.T
	db    *pgxpool.Pool
	query string
	args  []any
}

func (r pgxRow) Scan(dest ...any) {
	r.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.db.QueryRow(ctx, r.query, r.args...).Scan(dest...); err != nil {
		r.t.Fatalf("query row failed: %v\nquery: %s", err, r.query)
	}
}

func mustResetIAMState(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	statements := []string{
		`DELETE FROM iam.refresh_tokens`,
		`DELETE FROM iam.device_challenges`,
		`DELETE FROM iam.devices`,
		`DELETE FROM iam.recovery_codes`,
		`DELETE FROM iam.mfa_settings`,
		`DELETE FROM iam.role_permissions`,
		`DELETE FROM iam.user_roles`,
		`DELETE FROM iam.oauth_grants`,
		`DELETE FROM iam.roles`,
		`DELETE FROM iam.permissions`,
		`DELETE FROM iam.password_histories`,
		`DELETE FROM iam.user_profiles`,
		`DELETE FROM iam.users`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset state with %q: %v", stmt, err)
		}
	}
}
