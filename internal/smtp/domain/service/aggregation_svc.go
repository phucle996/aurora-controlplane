package smtp_domainsvc

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type AggregationService interface {
	GetWorkspaceAggregation(ctx context.Context, workspaceID string) (*entity.SMTPOverview, error)
}
