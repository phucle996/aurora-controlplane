package core_domainrepo

import (
	"context"
	"controlplane/internal/core/domain/entity"
)

type ZoneRepository interface {
	ListZones(ctx context.Context) ([]*entity.Zone, error)
	GetZone(ctx context.Context, id string) (*entity.Zone, error)
	GetZoneBySlug(ctx context.Context, slug string) (*entity.Zone, error)
	CreateZone(ctx context.Context, zone *entity.Zone) error
	UpdateZoneDescription(ctx context.Context, id, description string) (*entity.Zone, error)
	DeleteZone(ctx context.Context, id string) error
	CountDataPlanesByZoneID(ctx context.Context, zoneID string) (int64, error)
}
