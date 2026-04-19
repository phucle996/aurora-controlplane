package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
	registerStreamName  = "stream:mail:outgoing"
	registerBitmapUser  = "iam:register:bitmap:username"
	registerBitmapEmail = "iam:register:bitmap:email"
	registerTokenPrefix = "iam:ott:register:"
)

// AuthService implements auth use cases.
type AuthService struct {
	repo      iam_domainrepo.UserRepository
	deviceSvc iam_domainsvc.DeviceService
	tokenSvc  iam_domainsvc.TokenService
	mfaSvc    iam_domainsvc.MfaService
	rdb       *redis.Client
	cfg       *config.Config
}

func NewAuthService(
	repo iam_domainrepo.UserRepository,
	deviceSvc iam_domainsvc.DeviceService,
	tokenSvc iam_domainsvc.TokenService,
	mfaSvc iam_domainsvc.MfaService,
	rdb *redis.Client,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		repo:      repo,
		deviceSvc: deviceSvc,
		tokenSvc:  tokenSvc,
		mfaSvc:    mfaSvc,
		rdb:       rdb,
		cfg:       cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, user *entity.User, profile *entity.UserProfile, rawPassword string) error {
	// 1. Validate required inputs and normalize the identity fields.
	now := time.Now().UTC()
	if s == nil || s.repo == nil || s.cfg == nil {
		return iam_errorx.ErrRegistrationFailed
	}
	if user == nil || profile == nil {
		return iam_errorx.ErrRegistrationFailed
	}

	user.Username = id.NormalizeKey(user.Username)
	user.Email = id.NormalizeKey(user.Email)

	if user.Username == "" || user.Email == "" || profile.Fullname == "" || rawPassword == "" {
		return iam_errorx.ErrRegistrationFailed
	}

	// 2. Fast duplicate precheck via Redis bitmap; DB unique constraints remain final truth.
	if s.rdb != nil {
		usernameHit, err := s.rdb.GetBit(ctx, registerBitmapUser, id.BitmapIndex(user.Username)).Result()
		if err != nil {
			return fmt.Errorf("%w: %v", iam_errorx.ErrRegistrationFailed, err)
		}
		if usernameHit == 1 {
			if _, err := s.repo.GetByUsername(ctx, user.Username); err == nil {
				return iam_errorx.ErrUsernameAlreadyExists
			} else if !errors.Is(err, iam_errorx.ErrUserNotFound) {
				return err
			}
		}

		emailHit, err := s.rdb.GetBit(ctx, registerBitmapEmail, id.BitmapIndex(user.Email)).Result()
		if err != nil {
			return fmt.Errorf("%w: %v", iam_errorx.ErrRegistrationFailed, err)
		}
		if emailHit == 1 {
			if _, err := s.repo.GetByEmail(ctx, user.Email); err == nil {
				return iam_errorx.ErrEmailAlreadyExists
			} else if !errors.Is(err, iam_errorx.ErrUserNotFound) {
				return err
			}
		}
	}

	// 3. Hash password and generate application IDs.
	passwordHash, err := security.HashPassword(rawPassword)
	if err != nil {
		return err
	}

	userID, err := id.Generate()
	if err != nil {
		return err
	}

	profileID, err := id.Generate()
	if err != nil {
		return err
	}

	// 4. Stamp entity values for the pending account.
	user.PasswordHash = passwordHash
	user.SecurityLevel = 4
	user.CreatedAt = now
	user.UpdatedAt = now
	user.ID = userID

	profile.CreatedAt = now
	profile.UpdatedAt = now
	profile.ID = profileID
	profile.UserID = user.ID

	// 5. Persist user and profile in one repository transaction.
	if err := s.repo.CreatePendingAccount(ctx, user, profile); err != nil {
		return err
	}

	// 6. Mark the bitmap cache after a successful insert.
	if s.rdb != nil {
		_, _ = s.rdb.SetBit(ctx, registerBitmapUser, id.BitmapIndex(user.Username), 1).Result()
		_, _ = s.rdb.SetBit(ctx, registerBitmapEmail, id.BitmapIndex(user.Email), 1).Result()
	}

	return s.enqueueActivationEmail(ctx, user.ID, user.Email, profile.Fullname, now)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*entity.LoginResult, error) {

	// 1. Resolve the account by login identifier.
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, iam_errorx.ErrUserNotFound) {
			return nil, iam_errorx.ErrInvalidCredentials
		}
		return nil, err
	}

	// 2. Verify password before any account-state side effects.
	ok, err := security.VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, iam_errorx.ErrInvalidCredentials
	}

	now := time.Now().UTC()

	// 3. Pending accounts do not receive sessions; resend activation instead.
	if user.Status == "pending" {
		profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
		if err != nil {
			return nil, err
		}

		if err := s.enqueueActivationEmail(ctx, user.ID, user.Email, profile.Fullname, now); err != nil {
			return nil, err
		}

		return &entity.LoginResult{Pending: true}, nil
	}

	// 4. Reject non-active accounts.
	if user.Status != "active" {
		return nil, iam_errorx.ErrUserInactive
	}

	// 5. Resolve or create the logical device for this session.
	resolvedDevice, err := s.resolveLoginDevice(ctx, user)
	if err != nil {
		return nil, err
	}

	// 6. MFA Gate — if the user has any active MFA methods, issue a challenge
	//    instead of tokens. The client must complete MFA via POST /auth/mfa/verify.
	if s.mfaSvc != nil {
		required, challengeID, methods, err := s.mfaSvc.CheckAndChallenge(ctx, user.ID, resolvedDevice.ID)
		if err != nil {
			return nil, fmt.Errorf("login: mfa check: %w", err)
		}
		if required {
			return &entity.LoginResult{
				MFARequired:         true,
				MFAChallengeID:      challengeID,
				MFAAvailableMethods: methods,
			}, nil
		}
	}

	// 7. No MFA — delegate token issuance entirely to TokenService.
	tokenResult, err := s.tokenSvc.IssueAfterLogin(ctx, user, resolvedDevice)
	if err != nil {
		return nil, err
	}

	return &entity.LoginResult{
		AccessToken:           tokenResult.AccessToken,
		RefreshToken:          tokenResult.RefreshToken,
		DeviceID:              resolvedDevice.ID,
		AccessTokenExpiresAt:  tokenResult.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: tokenResult.RefreshTokenExpiresAt,
	}, nil
}

func (s *AuthService) enqueueActivationEmail(ctx context.Context, userID, email, fullName string, now time.Time) error {
	if s == nil || s.cfg == nil {
		return iam_errorx.ErrActivationFailed
	}
	if s.rdb == nil {
		return iam_errorx.ErrActivationFailed
	}

	token, err := security.GenerateToken(48, s.cfg.Security.OneTimeTokenSecret)
	if err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	verificationLink := fmt.Sprintf("http://localhost/api/v1/auth/activate?token=%s", url.QueryEscape(token))
	key := registerTokenPrefix + tokenDigest(token)
	ttl := s.cfg.Security.OneTimeTokenTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	values := map[string]any{
		"user_id":           userID,
		"email":             email,
		"full_name":         fullName,
		"purpose":           "verify_email",
		"verification_link": verificationLink,
		"created_at":        now.Format(time.RFC3339Nano),
		"expires_at":        now.Add(ttl).Format(time.RFC3339Nano),
	}

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, values)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	payload := map[string]any{
		"type":              "verify_email",
		"user_id":           userID,
		"email":             email,
		"full_name":         fullName,
		"template_key":      "verify-email",
		"verification_link": verificationLink,
		"created_at":        now.Format(time.RFC3339Nano),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrMailJobPublish, err)
	}

	if err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: registerStreamName,
		Values: map[string]any{
			"type":    "verify_email",
			"payload": string(raw),
		},
	}).Err(); err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrMailJobPublish, err)
	}

	return nil
}

func (s *AuthService) resolveLoginDevice(ctx context.Context, user *entity.User) (*entity.Device, error) {
	fingerprint := deriveLoginDeviceFingerprint(user.ID, user.Username)
	return s.deviceSvc.ResolveDevice(ctx, user.ID, fingerprint, "ES256")
}

func deriveLoginDeviceFingerprint(userID, username string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(userID) + "|" + strings.TrimSpace(username) + "|login"))
	return hex.EncodeToString(sum[:])
}

func (s *AuthService) Activate(ctx context.Context, token string) error {
	if s == nil || s.rdb == nil {
		return iam_errorx.ErrActivationFailed
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return iam_errorx.ErrActivationTokenInvalid
	}

	key := registerTokenPrefix + tokenDigest(token)
	values, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrActivationFailed, err)
	}
	if len(values) == 0 {
		return iam_errorx.ErrActivationTokenExpired
	}

	userID := strings.TrimSpace(values["user_id"])
	if userID == "" {
		return iam_errorx.ErrActivationTokenInvalid
	}

	if err := s.repo.Activate(ctx, userID); err != nil {
		return err
	}

	_ = s.rdb.Del(ctx, key).Err()
	return nil
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// ─── Password Reset ───────────────────────────────────────────────────────────

const (
	resetTokenPrefix     = "iam:ott:reset:"
	resetMailStream      = "stream:mail:outgoing"
	resetMailTemplateKey = "reset-password"
)

// ForgotPassword initiates a password-reset flow for the given email.
//
// Behaviour is deliberately identical whether the email exists or not — callers
// always receive a success response to prevent user enumeration.
// The actual reset link is delivered via the mail job stream.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	if s == nil || s.cfg == nil || s.rdb == nil {
		return iam_errorx.ErrResetFailed
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return iam_errorx.ErrResetFailed
	}

	// 1. Resolve user — silently succeed on miss to prevent enumeration.
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// Unknown email — return nil to prevent enumeration.
		return nil
	}

	// 2. Fetch the display name for the email template.
	profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
	if err != nil {
		return nil // degrade gracefully
	}

	now := time.Now().UTC()

	// 3. Generate a single-use opaque token.
	rawToken, err := security.GenerateToken(48, s.cfg.Security.OneTimeTokenSecret)
	if err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrTokenGeneration, err)
	}

	// 4. Build the reset link.
	baseURL := strings.TrimRight("localhost", "/")
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, url.QueryEscape(rawToken))

	// 5. Persist the token metadata in Redis with TTL.
	ttl := s.cfg.Security.OneTimeTokenTTL

	key := resetTokenPrefix + tokenDigest(rawToken)
	values := map[string]any{
		"user_id":    user.ID,
		"email":      email,
		"full_name":  profile.Fullname,
		"purpose":    "reset_password",
		"reset_link": resetLink,
		"created_at": now.Format(time.RFC3339Nano),
		"expires_at": now.Add(ttl).Format(time.RFC3339Nano),
	}

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, values)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("%w: store reset token: %v", iam_errorx.ErrTokenGeneration, err)
	}

	// 6. Enqueue the reset-password mail job.
	payload := map[string]any{
		"type":         "reset_password",
		"user_id":      user.ID,
		"email":        email,
		"full_name":    profile.Fullname,
		"template_key": resetMailTemplateKey,
		"reset_link":   resetLink,
		"created_at":   now.Format(time.RFC3339Nano),
		"expires_at":   now.Add(ttl).Format(time.RFC3339Nano),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: marshal reset payload: %v", iam_errorx.ErrMailJobPublish, err)
	}

	if err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: resetMailStream,
		Values: map[string]any{
			"type":    "reset_password",
			"payload": string(raw),
		},
	}).Err(); err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrMailJobPublish, err)
	}

	return nil
}

// ResetPassword validates the one-time reset token and replaces the user's password.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if s == nil || s.cfg == nil || s.rdb == nil || s.repo == nil {
		return iam_errorx.ErrResetFailed
	}

	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return iam_errorx.ErrResetTokenInvalid
	}

	if len(newPassword) < 8 {
		return iam_errorx.ErrWeakPassword
	}

	// 1. Load token metadata from Redis.
	key := resetTokenPrefix + tokenDigest(rawToken)
	values, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrResetFailed, err)
	}
	if len(values) == 0 {
		return iam_errorx.ErrResetTokenExpired
	}

	userID := strings.TrimSpace(values["user_id"])
	if userID == "" {
		return iam_errorx.ErrResetTokenInvalid
	}

	// 2. Verify TTL by checking expires_at (belt-and-suspenders; Redis TTL is primary).
	if expiresAt, ok := values["expires_at"]; ok {
		t, err := time.Parse(time.RFC3339Nano, expiresAt)
		if err == nil && time.Now().UTC().After(t) {
			_ = s.rdb.Del(ctx, key).Err()
			return iam_errorx.ErrResetTokenExpired
		}
	}

	// 3. Hash the new password.
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("%w: hash: %v", iam_errorx.ErrResetFailed, err)
	}

	// 4. Persist the new password hash.
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("%w: %v", iam_errorx.ErrResetFailed, err)
	}

	// 5. Consume the token — single-use.
	_ = s.rdb.Del(ctx, key).Err()

	return nil
}

// Logout adds the access token's JTI to the blacklist and revokes the refresh token.
func (s *AuthService) Logout(ctx context.Context, jti string, rawRefreshToken string) error {
	// 1. Blacklist the access token
	if jti != "" && s.rdb != nil {
		key := fmt.Sprintf("iam:blacklist:%s", jti)
		s.rdb.Set(ctx, key, "revoked", s.cfg.Security.AccessSecretTTL)
	}

	// 2. Revoke the refresh token
	if rawRefreshToken != "" {
		_ = s.tokenSvc.RevokeByRaw(ctx, rawRefreshToken) // best effort
	}

	return nil
}
