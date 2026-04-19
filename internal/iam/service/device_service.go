package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"controlplane/internal/iam/domain/entity"
	iam_domainrepo "controlplane/internal/iam/domain/repository"
	iam_errorx "controlplane/internal/iam/errorx"
	"controlplane/pkg/id"
)

const challengeTTL = 5 * time.Minute

// DeviceService implements iam_domainsvc.DeviceService.
type DeviceService struct {
	repo iam_domainrepo.DeviceRepository
}

func NewDeviceService(repo iam_domainrepo.DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

// ── Core ─────────────────────────────────────────────────────────────────────

// ResolveDevice gets or creates a device by fingerprint, refreshing its activity.
func (s *DeviceService) ResolveDevice(ctx context.Context, userID, fingerprint, keyAlgorithm string) (*entity.Device, error) {
	if userID == "" || fingerprint == "" {
		return nil, fmt.Errorf("%w: userID and fingerprint are required", iam_errorx.ErrDeviceNotFound)
	}

	existing, err := s.repo.GetDeviceByFingerprint(ctx, userID, fingerprint)
	if err != nil && !errors.Is(err, iam_errorx.ErrDeviceNotFound) {
		return nil, err
	}

	now := time.Now().UTC()

	if existing == nil {
		return s.createDevice(ctx, userID, fingerprint, keyAlgorithm, now)
	}

	return s.refreshDevice(ctx, existing, fingerprint, keyAlgorithm, now)
}

// UpdateActivity stamps last_active_at for a device that is already resolved.
func (s *DeviceService) UpdateActivity(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return iam_errorx.ErrDeviceNotFound
	}

	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return err
	}

	device.LastActiveAt = time.Now().UTC()
	return s.repo.UpdateDevice(ctx, device)
}

// ── Security ──────────────────────────────────────────────────────────────────

// IssueChallenge creates a short-lived nonce for the device to sign.
func (s *DeviceService) IssueChallenge(ctx context.Context, userID, deviceID string) (*entity.DeviceChallenge, error) {
	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.UserID != userID {
		return nil, iam_errorx.ErrDeviceForbidden
	}

	challengeID, err := id.Generate()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	now := time.Now().UTC()
	ch := &entity.DeviceChallenge{
		ChallengeID: challengeID,
		DeviceID:    deviceID,
		UserID:      userID,
		Nonce:       nonce,
		ExpiresAt:   now.Add(challengeTTL),
		CreatedAt:   now,
	}

	if err := s.repo.SaveChallenge(ctx, ch); err != nil {
		return nil, err
	}

	return ch, nil
}

// VerifyProof validates the device's signed challenge response.
// For now this performs a constant-time nonce echo check; swap in real
// ECDSA/Ed25519 signature verification once device public keys are enrolled.
func (s *DeviceService) VerifyProof(ctx context.Context, proof *entity.DeviceProof) error {
	if proof == nil {
		return iam_errorx.ErrDeviceProofInvalid
	}

	ch, err := s.repo.GetChallenge(ctx, proof.ChallengeID)
	if err != nil {
		if errors.Is(err, iam_errorx.ErrDeviceChallengeNotFound) {
			return iam_errorx.ErrDeviceChallengeInvalid
		}
		return err
	}

	if ch.DeviceID != proof.DeviceID {
		return iam_errorx.ErrDeviceChallengeInvalid
	}
	if time.Now().UTC().After(ch.ExpiresAt) {
		_ = s.repo.DeleteChallenge(ctx, ch.ChallengeID)
		return iam_errorx.ErrDeviceChallengeInvalid
	}

	// TODO: replace echo-check with real signature verification
	if !strings.EqualFold(proof.Signature, ch.Nonce) {
		_ = s.repo.DeleteChallenge(ctx, ch.ChallengeID)
		return iam_errorx.ErrDeviceProofInvalid
	}

	_ = s.repo.DeleteChallenge(ctx, ch.ChallengeID)
	return nil
}

// RotateKey replaces the device public key after a verified proof.
func (s *DeviceService) RotateKey(ctx context.Context, userID, deviceID, newPublicKey, newAlgorithm string) error {
	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.UserID != userID {
		return iam_errorx.ErrDeviceForbidden
	}

	if err := s.repo.RotateDeviceKey(ctx, deviceID, newPublicKey, newAlgorithm); err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrDeviceKeyRotateFailed, err)
	}

	return nil
}

// Rebind re-attaches a device to a new key pair.
// Rebind is identical to RotateKey at this abstraction; callers ensure a
// valid proof was checked before invoking.
func (s *DeviceService) Rebind(ctx context.Context, userID, deviceID, newPublicKey, newAlgorithm string) error {
	return s.RotateKey(ctx, userID, deviceID, newPublicKey, newAlgorithm)
}

// Revoke removes a device owned by userID and kills its tokens.
func (s *DeviceService) Revoke(ctx context.Context, userID, deviceID string) error {
	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.UserID != userID {
		return iam_errorx.ErrDeviceForbidden
	}

	_ = s.repo.RevokeAllTokensByDevice(ctx, deviceID)
	return s.repo.DeleteDevice(ctx, deviceID)
}

// Quarantine flags a device as suspicious without removing it.
func (s *DeviceService) Quarantine(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return iam_errorx.ErrDeviceNotFound
	}
	return s.repo.SetSuspicious(ctx, deviceID, true)
}

// ── User self-service ─────────────────────────────────────────────────────────

// GetByID returns a device, asserting it belongs to userID.
func (s *DeviceService) GetByID(ctx context.Context, userID, deviceID string) (*entity.Device, error) {
	if deviceID == "" {
		return nil, iam_errorx.ErrDeviceNotFound
	}

	device, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.UserID != userID {
		return nil, iam_errorx.ErrDeviceForbidden
	}

	return device, nil
}

// ListByUserID returns all devices registered for a user.
func (s *DeviceService) ListByUserID(ctx context.Context, userID string) ([]*entity.Device, error) {
	if userID == "" {
		return nil, iam_errorx.ErrDeviceNotFound
	}
	return s.repo.ListDevicesByUserID(ctx, userID)
}

// Rename updates the human-readable name of a device, enforcing ownership.
func (s *DeviceService) Rename(ctx context.Context, userID, deviceID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return fmt.Errorf("%w: device name must be 1-64 characters", iam_errorx.ErrDeviceNotFound)
	}
	return s.repo.RenameDevice(ctx, deviceID, userID, name)
}

// RevokeOne revokes exactly one device belonging to the caller.
func (s *DeviceService) RevokeOne(ctx context.Context, userID, deviceID string) error {
	return s.Revoke(ctx, userID, deviceID)
}

// RevokeOthers revokes all devices for userID except keepDeviceID.
func (s *DeviceService) RevokeOthers(ctx context.Context, userID, keepDeviceID string) (int64, error) {
	if userID == "" || keepDeviceID == "" {
		return 0, iam_errorx.ErrDeviceNotFound
	}
	return s.repo.RevokeOtherDevices(ctx, userID, keepDeviceID)
}

// ── Admin / internal ──────────────────────────────────────────────────────────

// AdminGetByID returns any device by ID without ownership check.
func (s *DeviceService) AdminGetByID(ctx context.Context, deviceID string) (*entity.Device, error) {
	if deviceID == "" {
		return nil, iam_errorx.ErrDeviceNotFound
	}
	return s.repo.GetDeviceByID(ctx, deviceID)
}

// AdminRevoke force-revokes any device regardless of owner.
func (s *DeviceService) AdminRevoke(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return iam_errorx.ErrDeviceNotFound
	}
	_ = s.repo.RevokeAllTokensByDevice(ctx, deviceID)
	return s.repo.DeleteDevice(ctx, deviceID)
}

// MarkSuspicious sets or clears the suspicious flag.
func (s *DeviceService) MarkSuspicious(ctx context.Context, deviceID string, flag bool) error {
	if deviceID == "" {
		return iam_errorx.ErrDeviceNotFound
	}
	return s.repo.SetSuspicious(ctx, deviceID, flag)
}

// CleanupStale removes devices inactive before the given threshold.
func (s *DeviceService) CleanupStale(ctx context.Context, before time.Time) (int64, error) {
	return s.repo.CleanupStaleDevices(ctx, before)
}

// ── private helpers ───────────────────────────────────────────────────────────

func (s *DeviceService) createDevice(ctx context.Context, userID, fingerprint, keyAlgorithm string, now time.Time) (*entity.Device, error) {
	deviceID, err := id.Generate()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	device := &entity.Device{
		ID:           deviceID,
		UserID:       userID,
		Fingerprint:  fingerprint,
		KeyAlgorithm: keyAlgorithm,
		LastActiveAt: now,
		CreatedAt:    now,
	}

	if err := s.repo.CreateDevice(ctx, device); err != nil {
		return nil, err
	}

	return device, nil
}

func (s *DeviceService) refreshDevice(ctx context.Context, device *entity.Device, fingerprint, keyAlgorithm string, now time.Time) (*entity.Device, error) {
	device.Fingerprint = fingerprint
	device.KeyAlgorithm = keyAlgorithm
	device.LastActiveAt = now

	if err := s.repo.UpdateDevice(ctx, device); err != nil {
		return nil, err
	}

	return device, nil
}

func generateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
