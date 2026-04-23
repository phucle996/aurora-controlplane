package repository

import (
	"context"
	"testing"
	"time"

	"controlplane/internal/iam/domain/entity"
)

func TestDeviceRepositoryGetDeviceByFingerprintReturnsLatestByActivity(t *testing.T) {
	db := mustOpenIAMRepositoryIntegrationDB(t)
	mustResetIAMState(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userID := "device-user-1"
	fingerprint := "device-fingerprint-1"

	mustExecIAM(t, db, `INSERT INTO iam.users (id, username, email, phone, password_hash, security_level, status, status_reason, created_at, updated_at)
		VALUES ($1, 'deviceuser', 'deviceuser@example.com', NULL, 'hash', 4, 'active', '', NOW(), NOW())`, userID)

	older := time.Now().UTC().Add(-2 * time.Hour)
	newer := time.Now().UTC().Add(-time.Hour)
	mustExecIAM(t, db, `INSERT INTO iam.devices (id, user_id, device_public_key, key_algorithm, fingerprint, device_name, last_active_at, created_at)
		VALUES ($1, $2, $3, 'ecdsa-p256', $4, NULL, $5, NOW())`,
		"device-old", userID, "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----", fingerprint, older)
	mustExecIAM(t, db, `INSERT INTO iam.devices (id, user_id, device_public_key, key_algorithm, fingerprint, device_name, last_active_at, created_at)
		VALUES ($1, $2, $3, 'ecdsa-p256', $4, NULL, $5, NOW())`,
		"device-new", userID, "-----BEGIN PUBLIC KEY-----\nMIIC\n-----END PUBLIC KEY-----", fingerprint, newer)

	repo := NewDeviceRepository(db)
	got, err := repo.GetDeviceByFingerprint(ctx, userID, fingerprint)
	if err != nil {
		t.Fatalf("get device by fingerprint: %v", err)
	}

	if got.ID != "device-new" {
		t.Fatalf("expected latest device row, got %q", got.ID)
	}
	if got.DevicePublicKey == "" {
		t.Fatalf("expected device public key to be loaded")
	}

	_ = entity.Device{}
}
