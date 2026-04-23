package iam_service

import (
	"context"
	"strings"
	"sync"

	"controlplane/internal/iam/domain/entity"
	iam_domainrepo "controlplane/internal/iam/domain/repository"
	"controlplane/internal/security"
	"controlplane/pkg/id"
)

const adminAPITokenLength = 48

type AdminAPITokenService struct {
	repo    iam_domainrepo.AdminAPITokenRepository
	secrets security.SecretProvider

	mu          sync.RWMutex
	validHashes map[string]struct{}
	cacheVer    int64
}

func NewAdminAPITokenService(repo iam_domainrepo.AdminAPITokenRepository, secrets security.SecretProvider) *AdminAPITokenService {
	return &AdminAPITokenService{
		repo:        repo,
		secrets:     secrets,
		validHashes: make(map[string]struct{}),
	}
}

func (s *AdminAPITokenService) EnsureBootstrapToken(ctx context.Context) (string, bool, error) {
	if s == nil || s.repo == nil || s.secrets == nil {
		return "", false, nil
	}

	hasAny, err := s.repo.HasAdminAPITokens(ctx)
	if err != nil {
		return "", false, err
	}
	if hasAny {
		return "", false, nil
	}

	active, err := s.secrets.GetActive(security.SecretFamilyAdminAPI)
	if err != nil {
		return "", false, err
	}
	s.syncCacheVersion(active.Version)

	token, err := security.GenerateToken(adminAPITokenLength, active.Value)
	if err != nil {
		return "", false, err
	}

	tokenHash, err := security.HashToken(token, active.Value)
	if err != nil {
		return "", false, err
	}

	tokenID, err := id.Generate()
	if err != nil {
		return "", false, err
	}

	if err := s.repo.CreateAdminAPIToken(ctx, &entity.AdminAPIToken{
		ID:        tokenID,
		TokenHash: tokenHash,
	}); err != nil {
		return "", false, err
	}

	s.cacheHash(tokenHash)
	return token, true, nil
}

func (s *AdminAPITokenService) Validate(ctx context.Context, token string) (bool, error) {
	if s == nil || s.repo == nil || s.secrets == nil {
		return false, nil
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}

	candidates, err := s.secrets.GetCandidates(security.SecretFamilyAdminAPI)
	if err != nil {
		return false, err
	}
	if len(candidates) == 0 {
		return false, security.ErrSecretUnavailable
	}
	s.syncCacheVersion(candidates[0].Version)

	for _, candidate := range candidates {
		tokenHash, err := security.HashToken(token, candidate.Value)
		if err != nil {
			return false, err
		}

		if s.isCachedHash(tokenHash) {
			return true, nil
		}

		exists, err := s.repo.ExistsAdminAPITokenHash(ctx, tokenHash)
		if err != nil {
			return false, err
		}
		if !exists {
			continue
		}

		s.cacheHash(tokenHash)
		return true, nil
	}

	return false, nil
}

func (s *AdminAPITokenService) isCachedHash(tokenHash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.validHashes[tokenHash]
	return ok
}

func (s *AdminAPITokenService) cacheHash(tokenHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.validHashes[tokenHash] = struct{}{}
}

func (s *AdminAPITokenService) syncCacheVersion(version int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cacheVer == version {
		return
	}
	s.cacheVer = version
	s.validHashes = make(map[string]struct{})
}
