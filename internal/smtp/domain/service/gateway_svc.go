package smtp_domainsvc

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type GatewayService interface {
	ListGatewayItems(ctx context.Context, workspaceID string) ([]*entity.GatewayListItem, error)
	GetGatewayDetail(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error)
	ListGateways(ctx context.Context, workspaceID string) ([]*entity.Gateway, error)
	GetGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.Gateway, error)
	UpdateGatewayTemplates(ctx context.Context, workspaceID, gatewayID string, templateIDs []string) (*entity.GatewayDetail, error)
	UpdateGatewayEndpoints(ctx context.Context, workspaceID, gatewayID string, endpointIDs []string) (*entity.GatewayDetail, error)
	StartGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error)
	DrainGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error)
	DisableGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error)
	CreateGateway(ctx context.Context, gateway *entity.Gateway) error
	UpdateGateway(ctx context.Context, gateway *entity.Gateway) error
	DeleteGateway(ctx context.Context, workspaceID, gatewayID string) error
}
