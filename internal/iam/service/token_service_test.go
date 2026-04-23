package iam_service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"controlplane/internal/config"
	"controlplane/internal/iam/domain/entity"
	iam_domainsvc "controlplane/internal/iam/domain/service"
	iam_errorx "controlplane/internal/iam/errorx"
	"controlplane/internal/security"

	"errors"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

type casTokenRepo struct {
	mu       sync.Mutex
	consumed bool
	token    *entity.RefreshToken
}

func TestTokenServiceRevokeAllByUser(t *testing.T) {
	t.Parallel()

	repo := &stubTokenRepo{}
	svc := &TokenService{tokenRepo: repo}

	if err := svc.RevokeAllByUser(context.Background(), "user-123"); err != nil {
		t.Fatalf("revoke all by user: %v", err)
	}
	if repo.revokedUserID != "user-123" {
		t.Fatalf("expected revoke target user-123, got %q", repo.revokedUserID)
	}
}

func TestTokenServiceCleanupExpiredDelegatesToRepo(t *testing.T) {
	t.Parallel()

	repo := &stubTokenRepo{deletedExpired: 7}
	svc := &TokenService{tokenRepo: repo}

	deleted, err := svc.CleanupExpired(context.Background())
	if err != nil {
		t.Fatalf("cleanup expired: %v", err)
	}
	if !repo.deleteExpiredBatchCalled {
		t.Fatalf("expected cleanup to call repository delete expired batch")
	}
	if repo.deleteExpiredBatchLimit != cleanupBatchSize {
		t.Fatalf("expected cleanup batch size %d, got %d", cleanupBatchSize, repo.deleteExpiredBatchLimit)
	}
	if deleted != 7 {
		t.Fatalf("expected 7 deleted rows, got %d", deleted)
	}
}

func TestTokenServiceCleanupExpiredRunsMultipleBatches(t *testing.T) {
	t.Parallel()

	repo := &stubTokenRepo{
		deleteExpiredBatchSeq: []int64{cleanupBatchSize, 200},
	}
	svc := &TokenService{tokenRepo: repo}

	deleted, err := svc.CleanupExpired(context.Background())
	if err != nil {
		t.Fatalf("cleanup expired: %v", err)
	}
	if repo.deleteExpiredBatchCalls != 2 {
		t.Fatalf("expected 2 batch calls, got %d", repo.deleteExpiredBatchCalls)
	}
	if deleted != cleanupBatchSize+200 {
		t.Fatalf("expected total deleted %d, got %d", cleanupBatchSize+200, deleted)
	}
}

func TestTokenServiceRotateRejectsUnboundDevice(t *testing.T) {
	ctx := context.Background()

	rawToken := "refresh-token-123"
	userRepo := &stubUserRepo{
		usersByID: map[string]*entity.User{
			"user-1": {
				ID:            "user-1",
				Role:          "user",
				Status:        "active",
				SecurityLevel: 4,
			},
		},
	}
	tokenRepo := &stubTokenRepo{
		consumeActiveOK: true,
		tokenByHash: &entity.RefreshToken{
			ID:        "token-1",
			DeviceID:  "device-1",
			UserID:    "user-1",
			TokenHash: "hash",
			ExpiresAt: time.Now().UTC().Add(time.Hour),
			CreatedAt: time.Now().UTC(),
		},
	}
	deviceRepo := &stubDeviceRepo{
		device: &entity.Device{
			ID:              "device-1",
			UserID:          "user-1",
			DevicePublicKey: "",
			KeyAlgorithm:    security.AlgECDSAP256,
		},
	}
	svc := &TokenService{
		tokenRepo:  tokenRepo,
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
		secrets: &fakeSecretProvider{
			active: security.SecretVersion{Value: "refresh-secret"},
		},
	}

	result, err := svc.Rotate(ctx, &iam_domainsvc.RotateRequest{
		RawRefreshToken: rawToken,
		DeviceID:        "device-1",
		JTI:             "jti-1",
		IssuedAt:        time.Now().UTC().Unix(),
		HTM:             "POST",
		HTU:             "https://controlplane.example.com/api/v1/auth/refresh",
		TokenHash:       "refresh-token-hash",
		Signature:       "signature",
	})
	if !errors.Is(err, iam_errorx.ErrRefreshDeviceUnbound) {
		t.Fatalf("expected unbound device error, got result=%v err=%v", result, err)
	}
	if tokenRepo.revokedTokenID != "" {
		t.Fatalf("expected refresh token not to be revoked when device is unbound")
	}
}

func TestTokenServiceRotateRejectsReplayJTI(t *testing.T) {
	ctx := context.Background()

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	rawToken := "refresh-token-123"
	userRepo := &stubUserRepo{
		usersByID: map[string]*entity.User{
			"user-1": {
				ID:            "user-1",
				Role:          "user",
				Status:        "active",
				SecurityLevel: 4,
			},
		},
	}
	tokenRepo := &stubTokenRepo{
		consumeActiveOK: true,
		tokenByHash: &entity.RefreshToken{
			ID:        "token-1",
			DeviceID:  "device-1",
			UserID:    "user-1",
			TokenHash: "hash",
			ExpiresAt: time.Now().UTC().Add(time.Hour),
			CreatedAt: time.Now().UTC(),
		},
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 keypair: %v", err)
	}

	deviceRepo := &stubDeviceRepo{
		device: &entity.Device{
			ID:              "device-1",
			UserID:          "user-1",
			DevicePublicKey: encodeEd25519PublicKeyPEM(pub),
			KeyAlgorithm:    security.AlgEd25519,
		},
	}
	svc := &TokenService{
		tokenRepo:  tokenRepo,
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
		rdb:        client,
		cfg: &config.Config{
			App: config.AppCfg{
				PublicURL: "https://controlplane.example.com",
			},
			Security: config.SecurityCfg{
				AccessSecretTTL: time.Minute,
				RefreshTokenTTL: time.Minute,
				DeviceActiveTTL: time.Minute,
			},
		},
		secrets: &fakeSecretProvider{
			active: security.SecretVersion{Value: "refresh-secret"},
		},
	}

	req := &iam_domainsvc.RotateRequest{
		RawRefreshToken: rawToken,
		DeviceID:        "device-1",
		JTI:             "jti-1",
		IssuedAt:        time.Now().UTC().Unix(),
		HTM:             "POST",
		HTU:             "https://controlplane.example.com/api/v1/auth/refresh",
		TokenHash:       security.HashRefreshToken(rawToken),
	}
	req.Signature = signDevicePayloadForTest(t, priv, req)

	firstResult, err := svc.Rotate(ctx, req)
	if err != nil {
		t.Fatalf("first rotate: %v", err)
	}
	if firstResult == nil || firstResult.AccessToken == "" {
		t.Fatalf("expected first rotation to succeed")
	}

	secondResult, err := svc.Rotate(ctx, req)
	if !errors.Is(err, iam_errorx.ErrRefreshSignatureReplay) {
		t.Fatalf("expected replay error, got result=%v err=%v", secondResult, err)
	}
}

func TestTokenServiceRotateReturnsInvalidWhenTokenAlreadyConsumed(t *testing.T) {
	ctx := context.Background()

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	rawToken := "refresh-token-123"
	userRepo := &stubUserRepo{
		usersByID: map[string]*entity.User{
			"user-1": {
				ID:            "user-1",
				Role:          "user",
				Status:        "active",
				SecurityLevel: 4,
			},
		},
	}
	tokenRepo := &stubTokenRepo{
		consumeActiveOK: false,
		tokenByHash: &entity.RefreshToken{
			ID:        "token-1",
			DeviceID:  "device-1",
			UserID:    "user-1",
			TokenHash: "hash",
			ExpiresAt: time.Now().UTC().Add(time.Hour),
			CreatedAt: time.Now().UTC(),
		},
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 keypair: %v", err)
	}
	deviceRepo := &stubDeviceRepo{
		device: &entity.Device{
			ID:              "device-1",
			UserID:          "user-1",
			DevicePublicKey: encodeEd25519PublicKeyPEM(pub),
			KeyAlgorithm:    security.AlgEd25519,
		},
	}
	svc := &TokenService{
		tokenRepo:  tokenRepo,
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
		rdb:        client,
		cfg: &config.Config{
			App: config.AppCfg{
				PublicURL: "https://controlplane.example.com",
			},
			Security: config.SecurityCfg{
				AccessSecretTTL: time.Minute,
				RefreshTokenTTL: time.Minute,
				DeviceActiveTTL: time.Minute,
			},
		},
		secrets: &fakeSecretProvider{
			active: security.SecretVersion{Value: "refresh-secret"},
		},
	}

	req := &iam_domainsvc.RotateRequest{
		RawRefreshToken: rawToken,
		DeviceID:        "device-1",
		JTI:             "jti-1",
		IssuedAt:        time.Now().UTC().Unix(),
		HTM:             "POST",
		HTU:             "https://controlplane.example.com/api/v1/auth/refresh",
		TokenHash:       security.HashRefreshToken(rawToken),
	}
	req.Signature = signDevicePayloadForTest(t, priv, req)

	result, err := svc.Rotate(ctx, req)
	if !errors.Is(err, iam_errorx.ErrRefreshTokenInvalid) {
		t.Fatalf("expected refresh token invalid for consumed token, got result=%v err=%v", result, err)
	}
}

func TestTokenServiceRotateConcurrentSameTokenOnlyOnePasses(t *testing.T) {
	ctx := context.Background()

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	rawToken := "refresh-token-123"
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 keypair: %v", err)
	}

	var repo casTokenRepo
	repo.token = &entity.RefreshToken{
		ID:        "token-1",
		DeviceID:  "device-1",
		UserID:    "user-1",
		TokenHash: "hash",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	tokenRepo := &repo

	userRepo := &stubUserRepo{
		usersByID: map[string]*entity.User{
			"user-1": {
				ID:            "user-1",
				Role:          "user",
				Status:        "active",
				SecurityLevel: 4,
			},
		},
	}
	deviceRepo := &stubDeviceRepo{
		device: &entity.Device{
			ID:              "device-1",
			UserID:          "user-1",
			DevicePublicKey: encodeEd25519PublicKeyPEM(pub),
			KeyAlgorithm:    security.AlgEd25519,
		},
	}

	svc := &TokenService{
		tokenRepo:  tokenRepo,
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
		rdb:        client,
		cfg: &config.Config{
			App: config.AppCfg{
				PublicURL: "https://controlplane.example.com",
			},
			Security: config.SecurityCfg{
				AccessSecretTTL: time.Minute,
				RefreshTokenTTL: time.Minute,
				DeviceActiveTTL: time.Minute,
			},
		},
		secrets: &fakeSecretProvider{
			active: security.SecretVersion{Value: "refresh-secret"},
		},
	}

	makeReq := func(jti string) *iam_domainsvc.RotateRequest {
		req := &iam_domainsvc.RotateRequest{
			RawRefreshToken: rawToken,
			DeviceID:        "device-1",
			JTI:             jti,
			IssuedAt:        time.Now().UTC().Unix(),
			HTM:             "POST",
			HTU:             "https://controlplane.example.com/api/v1/auth/refresh",
			TokenHash:       security.HashRefreshToken(rawToken),
		}
		req.Signature = signDevicePayloadForTest(t, priv, req)
		return req
	}

	reqA := makeReq("jti-a")
	reqB := makeReq("jti-b")

	type outcome struct {
		result *iam_domainsvc.TokenResult
		err    error
	}
	results := make(chan outcome, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		res, err := svc.Rotate(ctx, reqA)
		results <- outcome{result: res, err: err}
	}()
	go func() {
		defer wg.Done()
		res, err := svc.Rotate(ctx, reqB)
		results <- outcome{result: res, err: err}
	}()
	wg.Wait()
	close(results)

	successes := 0
	invalids := 0
	for out := range results {
		switch {
		case out.err == nil && out.result != nil && out.result.AccessToken != "":
			successes++
		case errors.Is(out.err, iam_errorx.ErrRefreshTokenInvalid):
			invalids++
		default:
			t.Fatalf("unexpected rotate outcome: result=%v err=%v", out.result, out.err)
		}
	}

	if successes != 1 || invalids != 1 {
		t.Fatalf("expected 1 success and 1 invalid, got success=%d invalid=%d", successes, invalids)
	}
}

func (r *casTokenRepo) Create(ctx context.Context, token *entity.RefreshToken) error { return nil }
func (r *casTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneRefreshToken(r.token), nil
}
func (r *casTokenRepo) Revoke(ctx context.Context, tokenID string) error { return nil }
func (r *casTokenRepo) ConsumeActive(ctx context.Context, tokenID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.consumed {
		return false, nil
	}
	r.consumed = true
	return true, nil
}
func (r *casTokenRepo) RevokeAllByDevice(ctx context.Context, deviceID string) error { return nil }
func (r *casTokenRepo) RevokeAllByUser(ctx context.Context, userID string) error     { return nil }
func (r *casTokenRepo) DeleteExpired(ctx context.Context) (int64, error)             { return 0, nil }
func (r *casTokenRepo) DeleteExpiredBatch(ctx context.Context, limit int64) (int64, error) {
	return 0, nil
}

func signDevicePayloadForTest(t *testing.T, priv ed25519.PrivateKey, req *iam_domainsvc.RotateRequest) string {
	t.Helper()

	payload := security.CanonicalRefreshPayload(req.JTI, req.IssuedAt, req.HTM, req.HTU, req.TokenHash, req.DeviceID)
	sum := sha256.Sum256([]byte(payload))
	sig := ed25519.Sign(priv, sum[:])
	return base64.RawURLEncoding.EncodeToString(sig)
}
