package smtp_domainrepo

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type ConsumerRepository interface {
	ListConsumerViewsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.ConsumerView, error)
	GetConsumerView(ctx context.Context, workspaceID, consumerID string) (*entity.ConsumerView, error)
	ListConsumerOptionsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.ConsumerOption, error)
	ListConsumersByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Consumer, error)
	GetConsumer(ctx context.Context, workspaceID, consumerID string) (*entity.Consumer, error)
	GetConsumerByID(ctx context.Context, consumerID string) (*entity.Consumer, error)
	CreateConsumer(ctx context.Context, consumer *entity.Consumer) error
	UpdateConsumer(ctx context.Context, consumer *entity.Consumer) error
	DeleteConsumer(ctx context.Context, workspaceID, consumerID string) error
}
