package smtp_domainrepo

import (
	"context"
	"time"

	"controlplane/internal/primitive/leaseassign"
	"controlplane/internal/primitive/rebalance"
	"controlplane/internal/smtp/domain/entity"
)

type RuntimeRepository interface {
	ListActivityLogs(ctx context.Context, workspaceID string) ([]*entity.ActivityLog, error)
	ListDeliveryAttempts(ctx context.Context, workspaceID string) ([]*entity.DeliveryAttempt, error)
	ListRuntimeHeartbeats(ctx context.Context) ([]*entity.RuntimeHeartbeat, error)
	ListGatewayAssignments(ctx context.Context) ([]*entity.GatewayShardAssignment, error)
	ListConsumerAssignments(ctx context.Context) ([]*entity.ConsumerShardAssignment, error)
	ListWorkspaceIDsByDataPlane(ctx context.Context, dataPlaneID string) ([]string, error)
	GetRuntimeVersionByDataPlane(ctx context.Context, dataPlaneID string) (int64, error)

	GetRuntimeDataPlane(ctx context.Context, dataPlaneID string) (*entity.RuntimeDataPlane, error)
	ListHealthyRuntimeDataPlanesByZone(ctx context.Context, zoneID string) ([]*entity.RuntimeDataPlane, error)
	UpsertRuntimeHeartbeat(ctx context.Context, heartbeat *entity.RuntimeHeartbeat) error
	ReplaceGatewayStatuses(ctx context.Context, dataPlaneID string, statuses []*entity.GatewayShardStatus) error
	ReplaceConsumerStatuses(ctx context.Context, dataPlaneID string, statuses []*entity.ConsumerShardStatus) error
	EnsureDesiredShards(ctx context.Context) error
	ReconcileAssignments(ctx context.Context) error
	ListGatewayWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error)
	ListConsumerWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error)
	ListGatewayAssignmentsForReconcile(ctx context.Context) ([]leaseassign.Assignment, error)
	ListConsumerAssignmentsForReconcile(ctx context.Context) ([]leaseassign.Assignment, error)
	ListHealthyRuntimeNodesByZone(ctx context.Context, zoneID string, now time.Time) ([]leaseassign.HealthyNode, error)
	ListGatewayRuntimeStatusByWork(ctx context.Context) (map[string]map[string]rebalance.RuntimeStatus, error)
	ListConsumerRuntimeStatusByWork(ctx context.Context) (map[string]map[string]rebalance.RuntimeStatus, error)
	ApplyGatewayAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error
	ApplyConsumerAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error

	ListGatewayAssignmentsByDataPlane(ctx context.Context, dataPlaneID string) ([]*entity.GatewayShardAssignment, error)
	ListConsumerAssignmentsByDataPlane(ctx context.Context, dataPlaneID string) ([]*entity.ConsumerShardAssignment, error)
}
