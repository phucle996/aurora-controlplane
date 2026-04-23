package smtp_domainrepo

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type EndpointRepository interface {
	ListEndpointViewsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.EndpointView, error)
	GetEndpointView(ctx context.Context, workspaceID, endpointID string) (*entity.EndpointView, error)
	ListEndpointsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Endpoint, error)
	GetEndpoint(ctx context.Context, workspaceID, endpointID string) (*entity.Endpoint, error)
	GetEndpointByID(ctx context.Context, endpointID string) (*entity.Endpoint, error)
	CreateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error
	UpdateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error
	DeleteEndpoint(ctx context.Context, workspaceID, endpointID string) error
}
