package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"controlplane/internal/core/domain/entity"
	core_domainrepo "controlplane/internal/core/domain/repository"
	"controlplane/internal/security"
	"controlplane/pkg/id"
)

const (
	defaultSecretRotationInterval = 168 * time.Hour
)

type secretFamilyCache struct {
	Active    security.SecretVersion
	Previous  *security.SecretVersion
	RotatedAt time.Time
	UpdatedAt time.Time
}

type SecretService struct {
	repo           core_domainrepo.SecretKeyVersionRepository
	masterKey      string
	rotateInterval time.Duration

	mu    sync.RWMutex
	cache map[string]secretFamilyCache
}

func NewSecretService(repo core_domainrepo.SecretKeyVersionRepository, masterKey string, rotateInterval time.Duration) *SecretService {
	if rotateInterval <= 0 {
		rotateInterval = defaultSecretRotationInterval
	}
	return &SecretService{
		repo:           repo,
		masterKey:      strings.TrimSpace(masterKey),
		rotateInterval: rotateInterval,
		cache:          make(map[string]secretFamilyCache, 4),
	}
}

func (s *SecretService) Bootstrap(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("core secret svc: repository is nil")
	}
	if s.masterKey == "" {
		return fmt.Errorf("core secret svc: master key is empty")
	}

	for _, family := range security.SecretFamilies() {
		if err := s.ensureFamilyActive(ctx, family); err != nil {
			return err
		}
	}

	for _, family := range security.SecretFamilies() {
		if err := s.reloadFamily(ctx, family); err != nil {
			return err
		}
	}

	return nil
}

func (s *SecretService) GetActive(family string) (security.SecretVersion, error) {
	if s == nil {
		return security.SecretVersion{}, security.ErrSecretUnavailable
	}
	family = strings.TrimSpace(family)
	if family == "" {
		return security.SecretVersion{}, security.ErrSecretUnavailable
	}

	entry, ok := s.readCacheEntry(family)
	if ok && strings.TrimSpace(entry.Active.Value) != "" {
		return entry.Active, nil
	}

	if err := s.reloadFamily(context.Background(), family); err != nil {
		return security.SecretVersion{}, security.ErrSecretUnavailable
	}

	entry, ok = s.readCacheEntry(family)
	if !ok || strings.TrimSpace(entry.Active.Value) == "" {
		return security.SecretVersion{}, security.ErrSecretUnavailable
	}

	return entry.Active, nil
}

func (s *SecretService) GetCandidates(family string) ([]security.SecretVersion, error) {
	if s == nil {
		return nil, security.ErrSecretUnavailable
	}
	family = strings.TrimSpace(family)
	if family == "" {
		return nil, security.ErrSecretUnavailable
	}

	entry, ok := s.readCacheEntry(family)
	if !ok || strings.TrimSpace(entry.Active.Value) == "" {
		if err := s.reloadFamily(context.Background(), family); err != nil {
			return nil, security.ErrSecretUnavailable
		}
		entry, ok = s.readCacheEntry(family)
		if !ok || strings.TrimSpace(entry.Active.Value) == "" {
			return nil, security.ErrSecretUnavailable
		}
	}

	out := make([]security.SecretVersion, 0, 2)
	out = append(out, entry.Active)
	if entry.Previous != nil && strings.TrimSpace(entry.Previous.Value) != "" {
		out = append(out, *entry.Previous)
	}
	return out, nil
}

func (s *SecretService) RotateDue(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("core secret svc: repository is nil")
	}

	now := time.Now().UTC()
	for _, family := range security.SecretFamilies() {
		active, err := s.GetActive(family)
		if err != nil {
			if err := s.reloadFamily(ctx, family); err != nil {
				return err
			}
			active, err = s.GetActive(family)
			if err != nil {
				return err
			}
		}
		if now.Before(active.ExpiresAt) {
			continue
		}
		if err := s.rotateFamily(ctx, family, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *SecretService) RefreshChangedFamilies(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("core secret svc: repository is nil")
	}

	latest, err := s.repo.ListLatestStatePerFamily(ctx)
	if err != nil {
		return err
	}

	for _, item := range latest {
		if item == nil || strings.TrimSpace(item.Family) == "" {
			continue
		}
		if s.needsRefresh(item) {
			if err := s.reloadFamily(ctx, item.Family); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SecretService) ensureFamilyActive(ctx context.Context, family string) error {
	rows, err := s.repo.GetFamilyVersions(ctx, family)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if row != nil && row.State == entity.SecretStateActive {
			return nil
		}
	}

	plainSecret, err := generateFamilySecret()
	if err != nil {
		return err
	}

	cipherText, err := security.EncryptSecret(plainSecret, s.masterKey)
	if err != nil {
		return err
	}

	secretID, err := id.Generate()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	return s.repo.CreateInitialActive(ctx, &entity.SecretKeyVersion{
		ID:               secretID,
		Family:           family,
		Version:          1,
		State:            entity.SecretStateActive,
		SecretCiphertext: cipherText,
		ExpiresAt:        now.Add(s.rotateInterval),
		RotatedAt:        now,
	})
}

func (s *SecretService) rotateFamily(ctx context.Context, family string, now time.Time) error {
	plainSecret, err := generateFamilySecret()
	if err != nil {
		return err
	}

	cipherText, err := security.EncryptSecret(plainSecret, s.masterKey)
	if err != nil {
		return err
	}

	nextID, err := id.Generate()
	if err != nil {
		return err
	}

	if err := s.repo.RotateFamilyTx(ctx, family, &entity.SecretKeyVersion{
		ID:               nextID,
		Family:           family,
		State:            entity.SecretStateActive,
		SecretCiphertext: cipherText,
		ExpiresAt:        now.Add(s.rotateInterval),
		RotatedAt:        now,
	}); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.cache, family)
	s.mu.Unlock()

	return s.reloadFamily(ctx, family)
}

func (s *SecretService) reloadFamily(ctx context.Context, family string) error {
	rows, err := s.repo.GetFamilyVersions(ctx, family)
	if err != nil {
		return err
	}

	var (
		active   *security.SecretVersion
		previous *security.SecretVersion
		updated  time.Time
		rotated  time.Time
	)
	for _, row := range rows {
		if row == nil {
			continue
		}
		plain, err := security.DecryptSecret(row.SecretCiphertext, s.masterKey)
		if err != nil {
			return err
		}
		current := security.SecretVersion{
			Family:    row.Family,
			Version:   row.Version,
			Value:     plain,
			ExpiresAt: row.ExpiresAt,
			RotatedAt: row.RotatedAt,
		}
		if row.State == entity.SecretStateActive {
			item := current
			active = &item
		}
		if row.State == entity.SecretStatePrevious && previous == nil {
			item := current
			previous = &item
		}
		if row.UpdatedAt.After(updated) {
			updated = row.UpdatedAt
		}
		if row.RotatedAt.After(rotated) {
			rotated = row.RotatedAt
		}
	}

	if active == nil {
		return security.ErrSecretUnavailable
	}

	s.mu.Lock()
	s.cache[family] = secretFamilyCache{
		Active:    *active,
		Previous:  previous,
		UpdatedAt: updated,
		RotatedAt: rotated,
	}
	s.mu.Unlock()
	return nil
}

func (s *SecretService) needsRefresh(value *entity.SecretFamilyState) bool {
	s.mu.RLock()
	cached, ok := s.cache[value.Family]
	s.mu.RUnlock()
	if !ok {
		return true
	}
	if value.UpdatedAt.After(cached.UpdatedAt) {
		return true
	}
	if value.RotatedAt.After(cached.RotatedAt) {
		return true
	}
	return false
}

func (s *SecretService) readCacheEntry(family string) (secretFamilyCache, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[family]
	return entry, ok
}

func generateFamilySecret() (string, error) {
	raw := make([]byte, 48)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("core secret svc: read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
