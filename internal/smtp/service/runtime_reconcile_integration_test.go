package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"controlplane/internal/app/bootstrap"
	smtp_repo "controlplane/internal/smtp/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRuntimeServiceReconcile_FailoverPromotesPendingInSameZone(t *testing.T) {
	db := mustOpenSMTPIntegrationDB(t)
	mustResetSMTPRuntimeState(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	zoneAID := "01AAAAAAAAAAAAAAAAAAAAAAAA"
	zoneBID := "01BBBBBBBBBBBBBBBBBBBBBBBB"
	gatewayID := "01CCCCCCCCCCCCCCCCCCCCCCCC"

	oldOwnerID := "01DDDDDDDDDDDDDDDDDDDDDDDD"
	newOwnerID := "01EEEEEEEEEEEEEEEEEEEEEEEE"
	crossZoneID := "01FFFFFFFFFFFFFFFFFFFFFFFF"

	mustExec(t, db, `INSERT INTO core.zones (id, name, slug) VALUES ($1, 'Zone A', 'zone-a')`, zoneAID)
	mustExec(t, db, `INSERT INTO core.zones (id, name, slug) VALUES ($1, 'Zone B', 'zone-b')`, zoneBID)

	mustExec(t, db, `
		INSERT INTO core.data_planes (id, node_key, name, zone_id, grpc_endpoint, version, cert_serial, status, last_seen_at, created_at, updated_at)
		VALUES ($1, 'node-old', 'old-owner', $2, 'grpc://old-owner', '', '', 'stale', NOW(), NOW(), NOW())
	`, oldOwnerID, zoneAID)
	mustExec(t, db, `
		INSERT INTO core.data_planes (id, node_key, name, zone_id, grpc_endpoint, version, cert_serial, status, last_seen_at, created_at, updated_at)
		VALUES ($1, 'node-new', 'new-owner', $2, 'grpc://new-owner', '', '', 'healthy', NOW(), NOW(), NOW())
	`, newOwnerID, zoneAID)
	mustExec(t, db, `
		INSERT INTO core.data_planes (id, node_key, name, zone_id, grpc_endpoint, version, cert_serial, status, last_seen_at, created_at, updated_at)
		VALUES ($1, 'node-cross', 'cross-zone-owner', $2, 'grpc://cross-zone', '', '', 'healthy', NOW(), NOW(), NOW())
	`, crossZoneID, zoneBID)

	mustExec(t, db, `INSERT INTO smtp.runtime_heartbeats (data_plane_id, sent_at, capacity, updated_at) VALUES ($1, NOW(), 1, NOW())`, newOwnerID)
	mustExec(t, db, `INSERT INTO smtp.runtime_heartbeats (data_plane_id, sent_at, capacity, updated_at) VALUES ($1, NOW(), 99, NOW())`, crossZoneID)

	mustExec(t, db, `
		INSERT INTO smtp.gateways (id, name, status, zone_id, desired_shard_count, created_at, updated_at)
		VALUES ($1, 'gateway-a', 'active', $2, 1, NOW(), NOW())
	`, gatewayID, zoneAID)
	mustExec(t, db, `
		INSERT INTO smtp.gateway_shards (gateway_id, shard_id, desired_state, created_at, updated_at)
		VALUES ($1, 0, 'active', NOW(), NOW())
	`, gatewayID)
	mustExec(t, db, `
		INSERT INTO smtp.gateway_shard_assignments (
			gateway_id, shard_id, data_plane_id, generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at
		) VALUES (
			$1, 0, $2, 10, 'active', 'active', NOW() + interval '30 second', NOW(), NOW()
		)
	`, gatewayID, oldOwnerID)

	runtimeRepo := smtp_repo.NewRuntimeRepository(db)
	runtimeSvc := NewRuntimeService(runtimeRepo, nil, nil, nil, nil, nil)

	if err := runtimeSvc.Reconcile(ctx); err != nil {
		t.Fatalf("reconcile #1: %v", err)
	}

	rows := mustListGatewayAssignmentRows(t, db, gatewayID, 0)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows after first reconcile (revoking+pending), got %d", len(rows))
	}
	assertHasAssignmentRow(t, rows, oldOwnerID, "revoking")
	assertHasAssignmentRow(t, rows, newOwnerID, "pending")
	assertNoAssignmentRow(t, rows, crossZoneID)

	if err := runtimeSvc.Reconcile(ctx); err != nil {
		t.Fatalf("reconcile #2: %v", err)
	}

	rows = mustListGatewayAssignmentRows(t, db, gatewayID, 0)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after second reconcile, got %d", len(rows))
	}
	if rows[0].DataPlaneID != newOwnerID || rows[0].AssignmentState != "active" {
		t.Fatalf("expected active assignment on new owner, got %+v", rows[0])
	}
}

type gatewayAssignmentRow struct {
	DataPlaneID     string
	AssignmentState string
}

func mustListGatewayAssignmentRows(t *testing.T, db *pgxpool.Pool, gatewayID string, shardID int) []gatewayAssignmentRow {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
		SELECT data_plane_id, assignment_state
		FROM smtp.gateway_shard_assignments
		WHERE gateway_id = $1 AND shard_id = $2
		ORDER BY data_plane_id ASC
	`, gatewayID, shardID)
	if err != nil {
		t.Fatalf("query gateway assignments: %v", err)
	}
	defer rows.Close()

	out := make([]gatewayAssignmentRow, 0)
	for rows.Next() {
		var row gatewayAssignmentRow
		if err := rows.Scan(&row.DataPlaneID, &row.AssignmentState); err != nil {
			t.Fatalf("scan gateway assignment row: %v", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate gateway assignment rows: %v", err)
	}
	return out
}

func assertHasAssignmentRow(t *testing.T, rows []gatewayAssignmentRow, dataPlaneID, state string) {
	t.Helper()
	for _, row := range rows {
		if row.DataPlaneID == dataPlaneID && row.AssignmentState == state {
			return
		}
	}
	t.Fatalf("expected assignment row data_plane_id=%s state=%s, got %+v", dataPlaneID, state, rows)
}

func assertNoAssignmentRow(t *testing.T, rows []gatewayAssignmentRow, dataPlaneID string) {
	t.Helper()
	for _, row := range rows {
		if row.DataPlaneID == dataPlaneID {
			t.Fatalf("expected no assignment for data_plane_id=%s, got %+v", dataPlaneID, rows)
		}
	}
}

func mustOpenSMTPIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("SMTP_RUNTIME_IT_DB_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("set SMTP_RUNTIME_IT_DB_URL (or TEST_DATABASE_URL) to run SMTP runtime integration tests")
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

func mustResetSMTPRuntimeState(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	statements := []string{
		`DELETE FROM smtp.consumer_runtime_statuses`,
		`DELETE FROM smtp.gateway_runtime_statuses`,
		`DELETE FROM smtp.consumer_assignments`,
		`DELETE FROM smtp.gateway_shard_assignments`,
		`DELETE FROM smtp.consumer_shards`,
		`DELETE FROM smtp.gateway_shards`,
		`DELETE FROM smtp.gateway_templates`,
		`DELETE FROM smtp.gateway_endpoints`,
		`DELETE FROM smtp.template_versions`,
		`DELETE FROM smtp.templates`,
		`DELETE FROM smtp.delivery_attempts`,
		`DELETE FROM smtp.activity_logs`,
		`DELETE FROM smtp.endpoint_secrets`,
		`DELETE FROM smtp.consumer_secrets`,
		`DELETE FROM smtp.endpoints`,
		`DELETE FROM smtp.gateways`,
		`DELETE FROM smtp.consumers`,
		`DELETE FROM smtp.runtime_heartbeats`,
		`DELETE FROM core.workspace_members`,
		`DELETE FROM core.workspaces`,
		`DELETE FROM core.tenant_members`,
		`DELETE FROM core.tenants`,
		`DELETE FROM core.data_planes`,
		`DELETE FROM core.zones`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt); err != nil {
			t.Fatalf("reset state with %q: %v", stmt, err)
		}
	}
}

func mustExec(t *testing.T, db *pgxpool.Pool, query string, args ...any) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.Exec(ctx, query, args...); err != nil {
		t.Fatalf("exec query failed: %v\nquery: %s", err, query)
	}
}
