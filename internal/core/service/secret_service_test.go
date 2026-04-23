package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"controlplane/internal/core/domain/entity"
	"controlplane/internal/security"
)

func TestSecretServiceBootstrapUsesCacheOnRead(t *testing.T) {
	t.Parallel()

	repo := newInMemorySecretRepo()
	svc := NewSecretService(repo, "12345678901234567890123456789012", time.Hour)

	if err := svc.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	before := repo.getFamilyVersionsCalls()
	for i := 0; i < 10; i++ {
		for _, family := range security.SecretFamilies() {
			active, err := svc.GetActive(family)
			if err != nil {
				t.Fatalf("get active %s: %v", family, err)
			}
			if active.Version != 1 {
				t.Fatalf("expected version 1 for %s, got %d", family, active.Version)
			}

			candidates, err := svc.GetCandidates(family)
			if err != nil {
				t.Fatalf("get candidates %s: %v", family, err)
			}
			if len(candidates) != 1 {
				t.Fatalf("expected 1 candidate for %s, got %d", family, len(candidates))
			}
		}
	}

	after := repo.getFamilyVersionsCalls()
	if after != before {
		t.Fatalf("expected cache-only reads, getFamilyVersions calls changed: before=%d after=%d", before, after)
	}
}

func TestSecretServiceRotateDueUpdatesCacheImmediately(t *testing.T) {
	t.Parallel()

	repo := newInMemorySecretRepo()
	svc := NewSecretService(repo, "12345678901234567890123456789012", time.Hour)

	if err := svc.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	repo.forceExpireActive(security.SecretFamilyAccess, time.Now().UTC().Add(-time.Minute))
	if err := svc.RefreshChangedFamilies(context.Background()); err != nil {
		t.Fatalf("refresh changed families: %v", err)
	}

	if err := svc.RotateDue(context.Background()); err != nil {
		t.Fatalf("rotate due: %v", err)
	}

	active, err := svc.GetActive(security.SecretFamilyAccess)
	if err != nil {
		t.Fatalf("get active after rotate: %v", err)
	}
	if active.Version != 2 {
		t.Fatalf("expected active version 2 after rotate, got %d", active.Version)
	}

	candidates, err := svc.GetCandidates(security.SecretFamilyAccess)
	if err != nil {
		t.Fatalf("get candidates after rotate: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates after rotate, got %d", len(candidates))
	}
	if candidates[1].Version != 1 {
		t.Fatalf("expected previous version 1, got %d", candidates[1].Version)
	}
}

func TestSecretServiceCacheMissFallbacksToDBAndRecaches(t *testing.T) {
	t.Parallel()

	repo := newInMemorySecretRepo()
	svc := NewSecretService(repo, "12345678901234567890123456789012", time.Hour)

	if err := svc.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	before := repo.getFamilyVersionsCalls()
	svc.mu.Lock()
	svc.cache = make(map[string]secretFamilyCache)
	svc.mu.Unlock()

	active, err := svc.GetActive(security.SecretFamilyAccess)
	if err != nil {
		t.Fatalf("get active after cache miss: %v", err)
	}
	if active.Version != 1 {
		t.Fatalf("expected version 1 after fallback reload, got %d", active.Version)
	}

	candidates, err := svc.GetCandidates(security.SecretFamilyAccess)
	if err != nil {
		t.Fatalf("get candidates after cache miss: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate after fallback reload, got %d", len(candidates))
	}

	after := repo.getFamilyVersionsCalls()
	if after <= before {
		t.Fatalf("expected db fallback on cache miss, calls before=%d after=%d", before, after)
	}
}

type inMemorySecretRepo struct {
	mu                    sync.Mutex
	values                map[string][]*entity.SecretKeyVersion
	getFamilyVersionCalls int
}

func newInMemorySecretRepo() *inMemorySecretRepo {
	return &inMemorySecretRepo{
		values: make(map[string][]*entity.SecretKeyVersion),
	}
}

func (r *inMemorySecretRepo) ListLatestStatePerFamily(_ context.Context) ([]*entity.SecretFamilyState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*entity.SecretFamilyState, 0, len(r.values))
	for family, rows := range r.values {
		var latest entity.SecretFamilyState
		latest.Family = family
		for _, row := range rows {
			if row == nil {
				continue
			}
			if row.RotatedAt.After(latest.RotatedAt) {
				latest.RotatedAt = row.RotatedAt
			}
			if row.UpdatedAt.After(latest.UpdatedAt) {
				latest.UpdatedAt = row.UpdatedAt
			}
		}
		out = append(out, &latest)
	}
	return out, nil
}

func (r *inMemorySecretRepo) GetFamilyVersions(_ context.Context, family string) ([]*entity.SecretKeyVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.getFamilyVersionCalls++
	rows := r.values[family]
	out := make([]*entity.SecretKeyVersion, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out, nil
}

func (r *inMemorySecretRepo) CreateInitialActive(_ context.Context, value *entity.SecretKeyVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if value == nil {
		return nil
	}
	rows := r.values[value.Family]
	for _, row := range rows {
		if row != nil && row.State == entity.SecretStateActive {
			return nil
		}
	}
	copied := *value
	now := time.Now().UTC()
	copied.CreatedAt = now
	copied.UpdatedAt = now
	r.values[value.Family] = append(r.values[value.Family], &copied)
	return nil
}

func (r *inMemorySecretRepo) RotateFamilyTx(_ context.Context, family string, nextActive *entity.SecretKeyVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows := r.values[family]
	now := time.Now().UTC()

	var (
		currentActive *entity.SecretKeyVersion
		rest          []*entity.SecretKeyVersion
	)
	for _, row := range rows {
		if row == nil {
			continue
		}
		if row.State == entity.SecretStateActive {
			c := *row
			currentActive = &c
			continue
		}
		if row.State == entity.SecretStatePrevious {
			continue
		}
		c := *row
		rest = append(rest, &c)
	}

	if currentActive != nil {
		currentActive.State = entity.SecretStatePrevious
		currentActive.UpdatedAt = now
		rest = append(rest, currentActive)
	}

	if nextActive != nil {
		c := *nextActive
		c.State = entity.SecretStateActive
		if currentActive != nil {
			c.Version = currentActive.Version + 1
		}
		c.CreatedAt = now
		c.UpdatedAt = now
		rest = append(rest, &c)
	}

	r.values[family] = rest
	return nil
}

func (r *inMemorySecretRepo) getFamilyVersionsCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getFamilyVersionCalls
}

func (r *inMemorySecretRepo) forceExpireActive(family string, expiry time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.values[family] {
		if row != nil && row.State == entity.SecretStateActive {
			row.ExpiresAt = expiry
			row.UpdatedAt = time.Now().UTC()
		}
	}
}
