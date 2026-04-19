package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"controlplane/internal/config"
	"controlplane/internal/iam/domain/entity"
	iam_domainrepo "controlplane/internal/iam/domain/repository"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	"controlplane/internal/security"
	"controlplane/pkg/id"

	"github.com/redis/go-redis/v9"
)

const (
	// refreshNonceWindow is the max age of a signed nonce we accept (5 minutes).
	// Prevents replay attacks with stale signatures.
	refreshNonceWindow = 5 * time.Minute
)

// TokenService implements iam_domainsvc.TokenService.
type TokenService struct {
	tokenRepo  iam_domainrepo.TokenRepository
	deviceRepo iam_domainrepo.DeviceRepository
	userRepo   iam_domainrepo.UserRepository
	rdb        *redis.Client
	cfg        *config.Config
}

func NewTokenService(
	tokenRepo iam_domainrepo.TokenRepository,
	deviceRepo iam_domainrepo.DeviceRepository,
	userRepo iam_domainrepo.UserRepository,
	rdb *redis.Client,
	cfg *config.Config,
) *TokenService {
	return &TokenService{
		tokenRepo:  tokenRepo,
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
		rdb:        rdb,
		cfg:        cfg,
	}
}

// ── Flow 1: IssueAfterLogin ───────────────────────────────────────────────────

// IssueAfterLogin generates a refresh token + access token for a newly
// authenticated user/device pair. Called exclusively by AuthService.Login.
func (s *TokenService) IssueAfterLogin(ctx context.Context, user *entity.User, device *entity.Device) (*iam_domainsvc.TokenResult, error) {
	if s == nil || s.cfg == nil {
		return nil, iam_errorx.ErrTokenGeneration
	}
	if user == nil || device == nil {
		return nil, iam_errorx.ErrTokenGeneration
	}

	now := time.Now().UTC()

	refreshRaw, refreshExpiry, err := s.issueRefreshToken(ctx, user, device, now)
	if err != nil {
		return nil, err
	}

	accessToken, accessExpiry, err := s.issueAccessToken(user, now)
	if err != nil {
		return nil, err
	}

	return &iam_domainsvc.TokenResult{
		AccessToken:           accessToken,
		RefreshToken:          refreshRaw,
		DeviceID:              device.ID,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

// IssueForMFA loads user+device from DB then issues a full token pair.
// Called by MfaHandler after MFA verification completes.
func (s *TokenService) IssueForMFA(ctx context.Context, userID, deviceID string) (*iam_domainsvc.TokenResult, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("token svc: get user for mfa: %w", err)
	}

	device, err := s.deviceRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("token svc: get device for mfa: %w", err)
	}

	return s.IssueAfterLogin(ctx, user, device)
}

// ── Flow 2: Rotate (client-signed proof) ─────────────────────────────────────

// Rotate validates the client's signed proof, revokes the presented refresh
// token, and issues a brand-new token pair.
//
// Security guarantees:
//  1. Token must exist, be non-revoked, non-expired in DB.
//  2. device_id inside the stored token must match req.DeviceID.
//  3. Client must produce a valid Ed25519/ECDSA signature over the canonical
//     payload using the device private key — server verifies with stored pubkey.
//  4. Nonce timestamp must be within refreshNonceWindow (replay protection).
//  5. Old token is revoked atomically before new tokens are issued.
func (s *TokenService) Rotate(ctx context.Context, req *iam_domainsvc.RotateRequest) (*iam_domainsvc.TokenResult, error) {
	if req == nil {
		return nil, iam_errorx.ErrRefreshTokenInvalid
	}

	rawToken := strings.TrimSpace(req.RawRefreshToken)
	deviceID := strings.TrimSpace(req.DeviceID)
	if rawToken == "" || deviceID == "" || req.Signature == "" {
		return nil, iam_errorx.ErrRefreshTokenInvalid
	}

	// 1. Timestamp freshness check (replay protection).
	reqTime := time.Unix(req.TimestampUnix, 0).UTC()
	now := time.Now().UTC()
	diff := now.Sub(reqTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > refreshNonceWindow {
		return nil, iam_errorx.ErrRefreshSignatureExpired
	}

	// 2. Hash raw token and look up in DB.
	hash, err := security.HashToken(rawToken, s.cfg.Security.RefreshTokenSecret)
	if err != nil {
		return nil, fmt.Errorf("%w: hash: %v", iam_errorx.ErrRefreshTokenInvalid, err)
	}

	stored, err := s.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, iam_errorx.ErrRefreshTokenInvalid) {
			return nil, iam_errorx.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("token svc: lookup: %w", err)
	}

	// 3. Assert device ownership.
	if stored.DeviceID != deviceID {
		return nil, iam_errorx.ErrRefreshTokenMismatch
	}

	// 4. Load device to get stored public key + algorithm.
	device, err := s.deviceRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("token svc: get device: %w", err)
	}

	if device.DevicePublicKey == "" {
		// Device has no enrolled key — cannot verify signature.
		return nil, iam_errorx.ErrRefreshSignatureInvalid
	}

	// 5. Reconstruct canonical payload and verify signature.
	payload := security.CanonicalRefreshPayload(rawToken, deviceID, req.Nonce, req.TimestampUnix)
	if err := security.VerifyDeviceSignature(
		device.DevicePublicKey,
		device.KeyAlgorithm,
		payload,
		req.Signature,
	); err != nil {
		return nil, iam_errorx.ErrRefreshSignatureInvalid
	}

	// 6. Load user for access token claims.
	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("token svc: get user: %w", err)
	}

	// 7. Revoke the consumed token (invalidate-then-reissue).
	if err := s.tokenRepo.Revoke(ctx, stored.ID); err != nil {
		return nil, fmt.Errorf("token svc: revoke old token: %w", err)
	}

	// 8. Stamp device activity.
	device.LastActiveAt = now
	_ = s.deviceRepo.UpdateDevice(ctx, device) // best-effort

	// 9. Issue new token pair.
	refreshRaw, refreshExpiry, err := s.issueRefreshToken(ctx, user, device, now)
	if err != nil {
		return nil, err
	}

	accessToken, accessExpiry, err := s.issueAccessToken(user, now)
	if err != nil {
		return nil, err
	}

	return &iam_domainsvc.TokenResult{
		AccessToken:           accessToken,
		RefreshToken:          refreshRaw,
		DeviceID:              device.ID,
		AccessTokenExpiresAt:  accessExpiry,
		RefreshTokenExpiresAt: refreshExpiry,
	}, nil
}

// ── private helpers ───────────────────────────────────────────────────────────

func (s *TokenService) issueRefreshToken(ctx context.Context, user *entity.User, device *entity.Device, now time.Time) (string, time.Time, error) {
	ttl := s.cfg.Security.RefreshTokenTTL
	if ttl <= 0 {
		ttl = 168 * time.Hour
	}

	rawToken, err := security.GenerateToken(64, s.cfg.Security.RefreshTokenSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: generate: %v", iam_errorx.ErrTokenGeneration, err)
	}

	hash, err := security.HashToken(rawToken, s.cfg.Security.RefreshTokenSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: hash: %v", iam_errorx.ErrTokenGeneration, err)
	}

	tokenID, err := id.Generate()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: id: %v", iam_errorx.ErrTokenGeneration, err)
	}

	rt := &entity.RefreshToken{
		ID:        tokenID,
		DeviceID:  device.ID,
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: now.Add(ttl),
		IsRevoked: false,
		CreatedAt: now,
	}

	if err := s.tokenRepo.Create(ctx, rt); err != nil {
		return "", time.Time{}, err
	}

	return rawToken, now.Add(ttl), nil
}

func (s *TokenService) issueAccessToken(user *entity.User, now time.Time) (string, time.Time, error) {
	ttl := s.cfg.Security.AccessSecretTTL

	tokenID, err := id.Generate()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	token, err := security.Sign(security.Claims{
		Subject:   user.ID,
		Role:      user.Role,
		Level:     int(user.SecurityLevel),
		Status:    user.Status,
		TokenID:   tokenID,
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}, s.cfg.Security.AccessSecretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	return token, now.Add(ttl), nil
}

// IsBlacklisted checks if the JTI is blacklisted in Redis.
func (s *TokenService) IsBlacklisted(ctx context.Context, jti string) bool {
	if jti == "" || s.rdb == nil {
		return false
	}
	key := fmt.Sprintf("iam:blacklist:%s", jti)
	val, err := s.rdb.Get(ctx, key).Result()
	return err == nil && val == "revoked"
}

// RevokeByRaw hashes the token and deletes it.
func (s *TokenService) RevokeByRaw(ctx context.Context, rawRefreshToken string) error {
	hash, err := security.HashToken(rawRefreshToken, s.cfg.Security.RefreshTokenSecret)
	if err != nil {
		return err
	}
	stored, err := s.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil
	}
	return s.tokenRepo.Revoke(ctx, stored.ID)
}
