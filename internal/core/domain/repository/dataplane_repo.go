package core_domainrepo

import (
	"context"
	"time"

	"controlplane/internal/core/domain/entity"
)

type DataPlaneRepository interface {
	SaveEnrollment(ctx context.Context, dataPlane *entity.DataPlane) (*entity.DataPlane, error)
	GetByID(ctx context.Context, id string) (*entity.DataPlane, error)
	GetByNodeKey(ctx context.Context, nodeKey string) (*entity.DataPlane, error)
	ListHealthyByZoneID(ctx context.Context, zoneID string) ([]*entity.DataPlane, error)
	UpdateHeartbeat(ctx context.Context, id, grpcEndpoint, version, status string, seenAt time.Time) error
	MarkStaleBefore(ctx context.Context, staleBefore time.Time) (int64, error)
}
