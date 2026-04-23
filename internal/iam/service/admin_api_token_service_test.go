package iam_service

import (
	"context"
	"testing"

	"controlplane/internal/iam/domain/entity"
	"controlplane/internal/security"
)

type adminAPITokenRepoStub struct {
	hasAny    bool
	hashes    map[string]struct{}
	existsHit int
}

func (r *adminAPITokenRepoStub) HasAdminAPITokens(ctx context.Context) (bool, error) {
	return r.hasAny, nil
}

func (r *adminAPITokenRepoStub) CreateAdminAPIToken(ctx context.Context, token *entity.AdminAPIToken) error {
	if r.hashes == nil {
		r.hashes = make(map[string]struct{})
	}
	r.hashes[token.TokenHash] = struct{}{}
	return nil
}

func (r *adminAPITokenRepoStub) ExistsAdminAPITokenHash(ctx context.Context, tokenHash string) (bool, error) {
	r.existsHit++
	_, ok := r.hashes[tokenHash]
	return ok, nil
}

func TestEnsureBootstrapTokenCachesHashOnly(t *testing.T) {
	ctx := context.Background()
	secret := "test-admin-secret-ensure"

	repo := &adminAPITokenRepoStub{
		hashes: make(map[string]struct{}),
	}
	svc := NewAdminAPITokenService(repo, &fakeSecretProvider{
		active: security.SecretVersion{
			Version: 1,
			Value:   secret,
		},
	})

	token, created, err := svc.EnsureBootstrapToken(ctx)
	if err != nil {
		t.Fatalf("EnsureBootstrapToken returned error: %v", err)
	}
	if !created {
		t.Fatalf("expected bootstrap token to be created")
	}
	if token == "" {
		t.Fatalf("expected plaintext token to be returned once")
	}

	tokenHash, err := security.HashToken(token, secret)
	if err != nil {
		t.Fatalf("hash token: %v", err)
	}

	if _, ok := svc.validHashes[tokenHash]; !ok {
		t.Fatalf("expected hash to be cached")
	}
	if _, ok := svc.validHashes[token]; ok {
		t.Fatalf("plaintext token must not be cached")
	}
}

func TestValidateCachesHashOnly(t *testing.T) {
	ctx := context.Background()
	secret := "test-admin-secret-validate"
	token := "candidate-admin-token"

	tokenHash, err := security.HashToken(token, secret)
	if err != nil {
		t.Fatalf("hash token: %v", err)
	}

	repo := &adminAPITokenRepoStub{
		hashes: map[string]struct{}{
			tokenHash: {},
		},
	}
	svc := NewAdminAPITokenService(repo, &fakeSecretProvider{
		active: security.SecretVersion{
			Version: 1,
			Value:   secret,
		},
	})

	ok, err := svc.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected token to validate")
	}
	if repo.existsHit != 1 {
		t.Fatalf("expected one repository check, got %d", repo.existsHit)
	}
	if _, exists := svc.validHashes[tokenHash]; !exists {
		t.Fatalf("expected token hash to be cached after validation")
	}
	if _, exists := svc.validHashes[token]; exists {
		t.Fatalf("plaintext token must not be cached")
	}

	ok, err = svc.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate second call returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected token to validate on cached path")
	}
	if repo.existsHit != 1 {
		t.Fatalf("expected cached validation without extra repository call, got %d", repo.existsHit)
	}
}
