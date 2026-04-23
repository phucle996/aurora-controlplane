package smtp_domainsvc

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type ConsumerService interface {
	ListConsumerViews(ctx context.Context, workspaceID string) ([]*entity.ConsumerView, error)
	GetConsumerView(ctx context.Context, workspaceID, consumerID string) (*entity.ConsumerView, error)
	ListConsumerOptions(ctx context.Context, workspaceID string) ([]*entity.ConsumerOption, error)
	ListConsumers(ctx context.Context, workspaceID string) ([]*entity.Consumer, error)
	GetConsumer(ctx context.Context, workspaceID, consumerID string) (*entity.Consumer, error)
	TryConnect(ctx context.Context, consumer *entity.Consumer) error
	CreateConsumer(ctx context.Context, consumer *entity.Consumer) error
	UpdateConsumer(ctx context.Context, consumer *entity.Consumer) error
	DeleteConsumer(ctx context.Context, workspaceID, consumerID string) error
}
