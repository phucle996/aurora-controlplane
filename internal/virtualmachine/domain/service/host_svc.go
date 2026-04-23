package service

import (
	"context"
	"time"

	"controlplane/internal/virtualmachine/domain/entity"
)

type HostService interface {
	ListHosts(ctx context.Context, filter entity.HostListFilter) (*entity.HostPage, error)
	GetHost(ctx context.Context, hostID string) (*entity.Host, error)
	GetHostBinding(ctx context.Context, hostID, agentID string) (*entity.HostBinding, error)
	UpsertHost(ctx context.Context, host *entity.Host) (*entity.Host, error)
	UpdateHostStatus(ctx context.Context, hostID, dataPlaneID, status string, lastSeenAt time.Time) (*entity.Host, error)
	ListHostOptions(ctx context.Context, filter entity.HostListFilter) ([]*entity.HostOption, error)
}
