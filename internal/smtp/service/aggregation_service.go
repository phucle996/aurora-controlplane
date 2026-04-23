package service

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
	smtp_domainrepo "controlplane/internal/smtp/domain/repository"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
)

type AggregationService struct {
	repo smtp_domainrepo.AggregationRepoInterface
}

func NewAggregationService(repo smtp_domainrepo.AggregationRepoInterface) smtp_domainsvc.AggregationService {
	return &AggregationService{repo: repo}
}

func (s *AggregationService) GetWorkspaceAggregation(ctx context.Context, workspaceID string) (*entity.SMTPOverview, error) {

	return s.repo.GetWorkspaceAggregation(ctx, workspaceID)
}
