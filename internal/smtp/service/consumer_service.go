package service

import (
	"context"
	"strings"

	"controlplane/internal/smtp/domain/entity"
	smtp_domainrepo "controlplane/internal/smtp/domain/repository"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
	"controlplane/pkg/id"
)

type ConsumerService struct {
	repo smtp_domainrepo.ConsumerRepository
}

func NewConsumerService(repo smtp_domainrepo.ConsumerRepository) smtp_domainsvc.ConsumerService {
	return &ConsumerService{repo: repo}
}

func (s *ConsumerService) ListConsumerViews(ctx context.Context, workspaceID string) ([]*entity.ConsumerView, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListConsumerViewsByWorkspace(ctx, workspaceID)
}

func (s *ConsumerService) GetConsumerView(ctx context.Context, workspaceID, consumerID string) (*entity.ConsumerView, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetConsumerView(ctx, workspaceID, strings.TrimSpace(consumerID))
}

func (s *ConsumerService) ListConsumerOptions(ctx context.Context, workspaceID string) ([]*entity.ConsumerOption, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListConsumerOptionsByWorkspace(ctx, workspaceID)
}

func (s *ConsumerService) ListConsumers(ctx context.Context, workspaceID string) ([]*entity.Consumer, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListConsumersByWorkspace(ctx, workspaceID)
}

func (s *ConsumerService) GetConsumer(ctx context.Context, workspaceID, consumerID string) (*entity.Consumer, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetConsumer(ctx, workspaceID, strings.TrimSpace(consumerID))
}

func (s *ConsumerService) TryConnect(ctx context.Context, consumer *entity.Consumer) error {
	if s == nil || consumer == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if err := normalizeConsumer(consumer); err != nil {
		return err
	}
	return probeConsumerConnection(ctx, consumer)
}

func (s *ConsumerService) CreateConsumer(ctx context.Context, consumer *entity.Consumer) error {
	if s == nil || s.repo == nil || consumer == nil {
		return smtp_errorx.ErrInvalidResource
	}

	if err := normalizeConsumer(consumer); err != nil {
		return err
	}

	consumerID, err := id.Generate()
	if err != nil {
		return err
	}
	consumer.ID = consumerID

	return s.repo.CreateConsumer(ctx, consumer)
}

func (s *ConsumerService) UpdateConsumer(ctx context.Context, consumer *entity.Consumer) error {
	if s == nil || s.repo == nil || consumer == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if strings.TrimSpace(consumer.ID) == "" {
		return smtp_errorx.ErrInvalidResource
	}
	if err := normalizeConsumer(consumer); err != nil {
		return err
	}
	return s.repo.UpdateConsumer(ctx, consumer)
}

func (s *ConsumerService) DeleteConsumer(ctx context.Context, workspaceID, consumerID string) error {
	if s == nil || s.repo == nil {
		return smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.DeleteConsumer(ctx, workspaceID, strings.TrimSpace(consumerID))
}

func normalizeConsumer(consumer *entity.Consumer) error {
	consumer.WorkspaceID = trimString(consumer.WorkspaceID)
	consumer.OwnerUserID = trimString(consumer.OwnerUserID)
	consumer.ZoneID = trimString(consumer.ZoneID)
	consumer.Name = trimString(consumer.Name)
	consumer.TransportType = trimString(consumer.TransportType)
	consumer.Source = trimString(consumer.Source)
	consumer.ConsumerGroup = trimString(consumer.ConsumerGroup)
	consumer.Status = defaultString(consumer.Status, "disabled")
	consumer.Note = trimString(consumer.Note)
	consumer.ConnectionConfig = normalizeJSON(consumer.ConnectionConfig)
	consumer.SecretConfig = normalizeJSON(consumer.SecretConfig)
	consumer.SecretRef = trimString(consumer.SecretRef)
	consumer.SecretProvider = defaultString(consumer.SecretProvider, "postgresql")
	consumer.WorkerConcurrency = maxInt(consumer.WorkerConcurrency, 1)
	consumer.AckTimeoutSeconds = maxInt(consumer.AckTimeoutSeconds, 1)
	consumer.BatchSize = maxInt(consumer.BatchSize, 1)
	consumer.DesiredShardCount = maxInt(consumer.DesiredShardCount, 1)

	if consumer.WorkspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	if consumer.ZoneID == "" {
		return smtp_errorx.ErrZoneRequired
	}
	if consumer.Name == "" || consumer.TransportType == "" || consumer.Source == "" || consumer.ConsumerGroup == "" {
		return smtp_errorx.ErrInvalidResource
	}
	return nil
}
