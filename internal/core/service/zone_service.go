package service

import (
	"context"
	"fmt"
	"strings"

	"controlplane/internal/core/domain/entity"
	core_domainrepo "controlplane/internal/core/domain/repository"
	core_errorx "controlplane/internal/core/errorx"
	"controlplane/pkg/id"
)

type ZoneService struct {
	repo core_domainrepo.ZoneRepository
}

func NewZoneService(repo core_domainrepo.ZoneRepository) *ZoneService {
	return &ZoneService{repo: repo}
}

func (s *ZoneService) ListZones(ctx context.Context) ([]*entity.Zone, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrZoneNotFound
	}
	return s.repo.ListZones(ctx)
}

func (s *ZoneService) GetZone(ctx context.Context, id string) (*entity.Zone, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrZoneNotFound
	}
	return s.repo.GetZone(ctx, id)
}

func (s *ZoneService) GetZoneBySlug(ctx context.Context, slug string) (*entity.Zone, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrZoneNotFound
	}
	return s.repo.GetZoneBySlug(ctx, normalizeSlug(slug))
}

func (s *ZoneService) CreateZone(ctx context.Context, zone *entity.Zone) error {
	if s == nil || s.repo == nil || zone == nil {
		return core_errorx.ErrZoneNotFound
	}

	zone.Name = strings.TrimSpace(zone.Name)
	zone.Slug = normalizeSlug(zone.Slug)
	zone.Description = strings.TrimSpace(zone.Description)
	if zone.Slug == "" {
		return fmt.Errorf("core svc: zone slug is empty")
	}

	zoneID, err := id.Generate()
	if err != nil {
		return err
	}

	zone.ID = zoneID
	return s.repo.CreateZone(ctx, zone)
}

func (s *ZoneService) UpdateZoneDescription(ctx context.Context, id, description string) (*entity.Zone, error) {
	if s == nil || s.repo == nil {
		return nil, core_errorx.ErrZoneNotFound
	}

	return s.repo.UpdateZoneDescription(ctx, id, strings.TrimSpace(description))
}

func (s *ZoneService) DeleteZone(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return core_errorx.ErrZoneNotFound
	}

	count, err := s.repo.CountDataPlanesByZoneID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return core_errorx.ErrZoneInUse
	}

	return s.repo.DeleteZone(ctx, id)
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	var builder strings.Builder
	builder.Grow(len(value))

	lastDash := false
	for _, ch := range value {
		isAlphaNum := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if isAlphaNum {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
