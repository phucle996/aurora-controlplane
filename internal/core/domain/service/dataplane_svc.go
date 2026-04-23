package core_domainsvc

import (
	"context"
	"controlplane/internal/core/domain/entity"
	"time"
)

type DataPlaneService interface {
	Enroll(ctx context.Context, dataPlane *entity.DataPlane, bootstrapToken, csrPEM string) (*entity.DataPlaneEnrollResult, error)
	Heartbeat(ctx context.Context, dataPlane *entity.DataPlane, peerDataPlaneID string) (*entity.DataPlaneHeartbeatResult, error)
	MarkStale(ctx context.Context, now time.Time) (int64, error)
	ListHealthyByZoneID(ctx context.Context, zoneID string) ([]*entity.DataPlane, error)
}
