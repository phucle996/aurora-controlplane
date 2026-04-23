package repository

import (
	"context"
	"time"

	"controlplane/internal/virtualmachine/domain/entity"
)

type HostRepository interface {
	ListHosts(ctx context.Context, filter entity.HostListFilter) (*entity.HostPage, error)
	GetHostByID(ctx context.Context, hostID string) (*entity.Host, error)
	GetHostByAgentID(ctx context.Context, agentID string) (*entity.Host, error)
	UpsertHost(ctx context.Context, host *entity.Host) (*entity.Host, error)
	UpdateHostStatus(ctx context.Context, hostID, dataPlaneID, status string, lastSeenAt time.Time) (*entity.Host, error)
	ListHostOptions(ctx context.Context, filter entity.HostListFilter) ([]*entity.HostOption, error)
}
