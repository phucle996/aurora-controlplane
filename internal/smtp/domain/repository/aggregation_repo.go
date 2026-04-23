package smtp_domainrepo

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type AggregationRepoInterface interface {
	GetWorkspaceAggregation(ctx context.Context, workspaceID string) (*entity.SMTPOverview, error)
}
