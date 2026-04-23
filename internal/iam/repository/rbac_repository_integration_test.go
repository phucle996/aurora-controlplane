package repository

import (
	"context"
	"testing"
	"time"
)

func TestRbacRepositoryListRoleEntriesAggregatesPermissions(t *testing.T) {
	db := mustOpenIAMRepositoryIntegrationDB(t)
	mustResetIAMState(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	mustExecIAM(t, db, `INSERT INTO iam.roles (id, name, level, description, created_at, updated_at)
		VALUES ('role-admin', 'admin', 0, 'Administrator', NOW(), NOW())`)
	mustExecIAM(t, db, `INSERT INTO iam.roles (id, name, level, description, created_at, updated_at)
		VALUES ('role-viewer', 'viewer', 10, 'Viewer', NOW(), NOW())`)

	mustExecIAM(t, db, `INSERT INTO iam.permissions (id, name, description, created_at)
		VALUES ('perm-read', 'iam:users:read', 'Read users', NOW())`)
	mustExecIAM(t, db, `INSERT INTO iam.permissions (id, name, description, created_at)
		VALUES ('perm-write', 'iam:users:write', 'Write users', NOW())`)

	mustExecIAM(t, db, `INSERT INTO iam.role_permissions (role_id, permission_id) VALUES ('role-admin', 'perm-read')`)
	mustExecIAM(t, db, `INSERT INTO iam.role_permissions (role_id, permission_id) VALUES ('role-admin', 'perm-write')`)
	mustExecIAM(t, db, `INSERT INTO iam.role_permissions (role_id, permission_id) VALUES ('role-viewer', 'perm-read')`)

	repo := NewRbacRepository(db)
	entries, err := repo.ListRoleEntries(ctx)
	if err != nil {
		t.Fatalf("list role entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two role entries, got %d", len(entries))
	}

	if entries[0].Role.Name != "admin" || len(entries[0].Permissions) != 2 {
		t.Fatalf("unexpected admin entry: %+v", entries[0])
	}
	if entries[1].Role.Name != "viewer" || len(entries[1].Permissions) != 1 {
		t.Fatalf("unexpected viewer entry: %+v", entries[1])
	}
}
