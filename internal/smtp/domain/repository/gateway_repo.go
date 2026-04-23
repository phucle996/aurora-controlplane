package smtp_domainrepo

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type GatewayRepository interface {
	ListGatewayItemsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.GatewayListItem, error)
	GetGatewayDetail(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error)
	ListGatewaysByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Gateway, error)
	GetGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.Gateway, error)
	GetGatewayByID(ctx context.Context, gatewayID string) (*entity.Gateway, error)
	CreateGateway(ctx context.Context, gateway *entity.Gateway) error
	UpdateGateway(ctx context.Context, gateway *entity.Gateway) error
	DeleteGateway(ctx context.Context, workspaceID, gatewayID string) error
}
