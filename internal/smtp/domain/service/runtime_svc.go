package smtp_domainsvc

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type RuntimeService interface {
	ListActivityLogs(ctx context.Context, workspaceID string) ([]*entity.ActivityLog, error)
	ListDeliveryAttempts(ctx context.Context, workspaceID string) ([]*entity.DeliveryAttempt, error)
	ListRuntimeHeartbeats(ctx context.Context) ([]*entity.RuntimeHeartbeat, error)
	ListGatewayAssignments(ctx context.Context) ([]*entity.GatewayShardAssignment, error)
	ListConsumerAssignments(ctx context.Context) ([]*entity.ConsumerShardAssignment, error)
	Sync(ctx context.Context, req *entity.RuntimeSyncRequest) (*entity.RuntimeSyncResponse, error)
	Report(ctx context.Context, req *entity.RuntimeReportRequest) (*entity.RuntimeReportResponse, error)
	Reconcile(ctx context.Context) error
}
