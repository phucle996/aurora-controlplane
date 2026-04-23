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

func BenchmarkUserRepositoryGetWhoAmI(b *testing.B) {
	db := openBenchIAMDB(b)
	resetIAMStateForBench(b, db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := "bench-whoami-user"
	roleAdminID := "bench-role-admin"
	roleViewerID := "bench-role-viewer"
	permReadID := "bench-perm-read"
	permWriteID := "bench-perm-write"

	execBenchIAM(b, db, `INSERT INTO iam.users (id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at)
		VALUES ($1, 'bench-user', 'bench@example.com', NULL, 'hash', 2, 'active', '', NOW(), NOW())`, userID)
	execBenchIAM(b, db, `INSERT INTO iam.user_profiles (id, user_id, fullname, avatar_url, bio, timezone, created_at, updated_at)
		VALUES ('bench-profile', $1, 'Bench User', '', '', 'UTC', NOW(), NOW())`, userID)
	execBenchIAM(b, db, `INSERT INTO iam.roles (id, name, level, description, created_at, updated_at)
		VALUES ($1, 'bench-admin', 0, '', NOW(), NOW())`, roleAdminID)
	execBenchIAM(b, db, `INSERT INTO iam.roles (id, name, level, description, created_at, updated_at)
		VALUES ($1, 'bench-viewer', 10, '', NOW(), NOW())`, roleViewerID)
	execBenchIAM(b, db, `INSERT INTO iam.permissions (id, name, slug, description, created_at)
		VALUES ($1, 'bench:read', 'bench:read', '', NOW())`, permReadID)
	execBenchIAM(b, db, `INSERT INTO iam.permissions (id, name, slug, description, created_at)
		VALUES ($1, 'bench:write', 'bench:write', '', NOW())`, permWriteID)
	execBenchIAM(b, db, `INSERT INTO iam.user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleAdminID)
	execBenchIAM(b, db, `INSERT INTO iam.user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleViewerID)
	execBenchIAM(b, db, `INSERT INTO iam.role_permissions (role_id, permission_id) VALUES ($1, $2)`, roleAdminID, permReadID)
	execBenchIAM(b, db, `INSERT INTO iam.role_permissions (role_id, permission_id) VALUES ($1, $2)`, roleAdminID, permWriteID)
	execBenchIAM(b, db, `INSERT INTO iam.role_permissions (role_id, permission_id) VALUES ($1, $2)`, roleViewerID, permReadID)

	repo := NewUserRepository(db)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := repo.GetWhoAmI(ctx, userID)
		if err != nil {
			b.Fatalf("get whoami: %v", err)
		}
		if result == nil {
			b.Fatalf("expected whoami result")
		}
	}
}

func openBenchIAMDB(b *testing.B) *pgxpool.Pool {
	b.Helper()

	dsn := strings.TrimSpace(os.Getenv("IAM_IT_DB_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		b.Skip("set IAM_IT_DB_URL (or TEST_DATABASE_URL) to run IAM benchmark")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		b.Fatalf("open db pool: %v", err)
	}
	b.Cleanup(db.Close)

	if err := db.Ping(ctx); err != nil {
		b.Fatalf("ping db: %v", err)
	}
	if err := bootstrap.RunMigrations(ctx, db); err != nil {
		b.Fatalf("run migrations: %v", err)
	}

	return db
}

func execBenchIAM(b *testing.B, db *pgxpool.Pool, query string, args ...any) {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.Exec(ctx, query, args...); err != nil {
		b.Fatalf("exec query failed: %v\nquery: %s", err, query)
	}
}

func resetIAMStateForBench(b *testing.B, db *pgxpool.Pool) {
	b.Helper()

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
			b.Fatalf("reset state with %q: %v", stmt, err)
		}
	}
}
