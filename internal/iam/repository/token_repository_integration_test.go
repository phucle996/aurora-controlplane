package repository

import (
	"context"
	"testing"
	"time"

	"controlplane/internal/iam/domain/entity"
)

func TestTokenRepositoryRevokeAllByUser(t *testing.T) {
	db := mustOpenIAMRepositoryIntegrationDB(t)
	mustResetIAMState(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userA := "user-revoke-a"
	userB := "user-revoke-b"
	deviceA := "device-revoke-a"
	deviceB := "device-revoke-b"
	deviceC := "device-revoke-c"

	mustExecIAM(t, db, `INSERT INTO iam.users (id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, 4, 'active', '', NOW(), NOW())`,
		userA, "usera", "usera@example.com", "hash-a")
	mustExecIAM(t, db, `INSERT INTO iam.users (id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, 4, 'active', '', NOW(), NOW())`,
		userB, "userb", "userb@example.com", "hash-b")

	for _, row := range []struct {
		id     string
		userID string
	}{
		{id: deviceA, userID: userA},
		{id: deviceB, userID: userA},
		{id: deviceC, userID: userB},
	} {
		mustExecIAM(t, db, `INSERT INTO iam.devices (id, user_id, device_public_key, key_algorithm, fingerprint, device_name, last_active_at, created_at)
			VALUES ($1, $2, $3, 'ecdsa-p256', $4, NULL, NOW(), NOW())`,
			row.id, row.userID, "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----", "fingerprint-"+row.id)
	}

	tokenRepo := NewTokenRepository(db)
	now := time.Now().UTC()
	for _, row := range []struct {
		id       string
		deviceID string
		userID   string
	}{
		{id: "token-a1", deviceID: deviceA, userID: userA},
		{id: "token-a2", deviceID: deviceB, userID: userA},
		{id: "token-b1", deviceID: deviceC, userID: userB},
	} {
		if err := tokenRepo.Create(ctx, &entity.RefreshToken{
			ID:        row.id,
			DeviceID:  row.deviceID,
			UserID:    row.userID,
			TokenHash: "hash-" + row.id,
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("create token %s: %v", row.id, err)
		}
	}

	if err := tokenRepo.RevokeAllByUser(ctx, userA); err != nil {
		t.Fatalf("revoke all by user: %v", err)
	}

	var revokedA1, revokedA2, revokedB1 bool
	mustQueryRowIAM(t, db, `SELECT is_revoked FROM iam.refresh_tokens WHERE id = $1`, "token-a1").Scan(&revokedA1)
	mustQueryRowIAM(t, db, `SELECT is_revoked FROM iam.refresh_tokens WHERE id = $1`, "token-a2").Scan(&revokedA2)
	mustQueryRowIAM(t, db, `SELECT is_revoked FROM iam.refresh_tokens WHERE id = $1`, "token-b1").Scan(&revokedB1)

	if !revokedA1 || !revokedA2 {
		t.Fatalf("expected userA tokens revoked, got token-a1=%v token-a2=%v", revokedA1, revokedA2)
	}
	if revokedB1 {
		t.Fatalf("expected userB token to remain active")
	}
}

func TestTokenRepositoryConsumeActiveCAS(t *testing.T) {
	db := mustOpenIAMRepositoryIntegrationDB(t)
	mustResetIAMState(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userID := "user-consume-1"
	deviceID := "device-consume-1"
	tokenID := "token-consume-1"

	mustExecIAM(t, db, `INSERT INTO iam.users (id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, 4, 'active', '', NOW(), NOW())`,
		userID, "consume-user", "consume@example.com", "hash")
	mustExecIAM(t, db, `INSERT INTO iam.devices (id, user_id, device_public_key, key_algorithm, fingerprint, device_name, last_active_at, created_at)
		VALUES ($1, $2, $3, 'ecdsa-p256', $4, NULL, NOW(), NOW())`,
		deviceID, userID, "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----", "fingerprint-"+deviceID)

	tokenRepo := NewTokenRepository(db)
	if err := tokenRepo.Create(ctx, &entity.RefreshToken{
		ID:        tokenID,
		DeviceID:  deviceID,
		UserID:    userID,
		TokenHash: "hash-" + tokenID,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	first, err := tokenRepo.ConsumeActive(ctx, tokenID)
	if err != nil {
		t.Fatalf("consume first call: %v", err)
	}
	second, err := tokenRepo.ConsumeActive(ctx, tokenID)
	if err != nil {
		t.Fatalf("consume second call: %v", err)
	}

	if !first {
		t.Fatalf("expected first consume to succeed")
	}
	if second {
		t.Fatalf("expected second consume to fail (already consumed)")
	}
}

func TestTokenRepositoryDeleteExpiredBatch(t *testing.T) {
	db := mustOpenIAMRepositoryIntegrationDB(t)
	mustResetIAMState(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userID := "user-batch-1"
	deviceID := "device-batch-1"

	mustExecIAM(t, db, `INSERT INTO iam.users (id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, $4, 4, 'active', '', NOW(), NOW())`,
		userID, "batch-user", "batch@example.com", "hash")
	mustExecIAM(t, db, `INSERT INTO iam.devices (id, user_id, device_public_key, key_algorithm, fingerprint, device_name, last_active_at, created_at)
		VALUES ($1, $2, $3, 'ecdsa-p256', $4, NULL, NOW(), NOW())`,
		deviceID, userID, "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----", "fingerprint-"+deviceID)

	tokenRepo := NewTokenRepository(db)
	now := time.Now().UTC()
	for _, row := range []struct {
		id        string
		expiresAt time.Time
	}{
		{id: "token-expired-1", expiresAt: now.Add(-3 * time.Hour)},
		{id: "token-expired-2", expiresAt: now.Add(-2 * time.Hour)},
		{id: "token-expired-3", expiresAt: now.Add(-1 * time.Hour)},
		{id: "token-active-1", expiresAt: now.Add(2 * time.Hour)},
	} {
		if err := tokenRepo.Create(ctx, &entity.RefreshToken{
			ID:        row.id,
			DeviceID:  deviceID,
			UserID:    userID,
			TokenHash: "hash-" + row.id,
			ExpiresAt: row.expiresAt,
			CreatedAt: now,
		}); err != nil {
			t.Fatalf("create token %s: %v", row.id, err)
		}
	}

	deleted1, err := tokenRepo.DeleteExpiredBatch(ctx, 2)
	if err != nil {
		t.Fatalf("delete batch 1: %v", err)
	}
	deleted2, err := tokenRepo.DeleteExpiredBatch(ctx, 2)
	if err != nil {
		t.Fatalf("delete batch 2: %v", err)
	}
	deleted3, err := tokenRepo.DeleteExpiredBatch(ctx, 2)
	if err != nil {
		t.Fatalf("delete batch 3: %v", err)
	}

	if deleted1 != 2 {
		t.Fatalf("expected first batch delete 2 rows, got %d", deleted1)
	}
	if deleted2 != 1 {
		t.Fatalf("expected second batch delete 1 row, got %d", deleted2)
	}
	if deleted3 != 0 {
		t.Fatalf("expected third batch delete 0 rows, got %d", deleted3)
	}

	var remaining int
	mustQueryRowIAM(t, db, `SELECT COUNT(*) FROM iam.refresh_tokens`).Scan(&remaining)
	if remaining != 1 {
		t.Fatalf("expected only active token to remain, got %d rows", remaining)
	}
}
