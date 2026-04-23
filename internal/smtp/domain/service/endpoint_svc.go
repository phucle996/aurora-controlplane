package smtp_domainsvc

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type EndpointService interface {
	ListEndpointViews(ctx context.Context, workspaceID string) ([]*entity.EndpointView, error)
	GetEndpointView(ctx context.Context, workspaceID, endpointID string) (*entity.EndpointView, error)
	ListEndpoints(ctx context.Context, workspaceID string) ([]*entity.Endpoint, error)
	GetEndpoint(ctx context.Context, workspaceID, endpointID string) (*entity.Endpoint, error)
	TryConnect(ctx context.Context, endpoint *entity.Endpoint) error
	CreateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error
	UpdateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error
	DeleteEndpoint(ctx context.Context, workspaceID, endpointID string) error
}
