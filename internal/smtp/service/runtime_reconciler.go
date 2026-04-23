package service

import (
	"context"
	"time"

	"controlplane/internal/primitive/leaseassign"
	"controlplane/internal/primitive/rebalance"
	smtp_domainrepo "controlplane/internal/smtp/domain/repository"
)

const (
	smtpAssignmentLeaseTTL = 30 * time.Second
	smtpHandoverGrace      = 45 * time.Second
)

type runtimeNodeProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p runtimeNodeProvider) ListHealthyNodesByZone(ctx context.Context, zoneID string, now time.Time) ([]leaseassign.HealthyNode, error) {
	return p.repo.ListHealthyRuntimeNodesByZone(ctx, zoneID, now)
}

type gatewayWorkProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p gatewayWorkProvider) ListWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error) {
	return p.repo.ListGatewayWorkShards(ctx)
}

type consumerWorkProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p consumerWorkProvider) ListWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error) {
	return p.repo.ListConsumerWorkShards(ctx)
}

type gatewayAssignmentProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p gatewayAssignmentProvider) ListAssignments(ctx context.Context) ([]leaseassign.Assignment, error) {
	return p.repo.ListGatewayAssignmentsForReconcile(ctx)
}

func (p gatewayAssignmentProvider) ApplyAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error {
	return p.repo.ApplyGatewayAssignments(ctx, rowsByWork)
}

type consumerAssignmentProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p consumerAssignmentProvider) ListAssignments(ctx context.Context) ([]leaseassign.Assignment, error) {
	return p.repo.ListConsumerAssignmentsForReconcile(ctx)
}

func (p consumerAssignmentProvider) ApplyAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error {
	return p.repo.ApplyConsumerAssignments(ctx, rowsByWork)
}

type gatewayStatusProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p gatewayStatusProvider) ListRuntimeStatusByWork(ctx context.Context) (map[string]map[string]rebalance.RuntimeStatus, error) {
	return p.repo.ListGatewayRuntimeStatusByWork(ctx)
}

type consumerStatusProvider struct {
	repo smtp_domainrepo.RuntimeRepository
}

func (p consumerStatusProvider) ListRuntimeStatusByWork(ctx context.Context) (map[string]map[string]rebalance.RuntimeStatus, error) {
	return p.repo.ListConsumerRuntimeStatusByWork(ctx)
}

func newGatewayCoordinator(runtimeRepo smtp_domainrepo.RuntimeRepository, projection rebalance.ProjectionSink) *rebalance.Coordinator {
	return &rebalance.Coordinator{
		RuntimeKey:       "smtp:gateway",
		WorkProvider:     gatewayWorkProvider{repo: runtimeRepo},
		NodeProvider:     runtimeNodeProvider{repo: runtimeRepo},
		AssignmentSource: gatewayAssignmentProvider{repo: runtimeRepo},
		StatusProvider:   gatewayStatusProvider{repo: runtimeRepo},
		Projection:       projection,
		Policy:           leaseassign.StickyLeastLoadedPolicy{},
		Config: rebalance.Config{
			AssignmentLeaseTTL: smtpAssignmentLeaseTTL,
			HandoverGrace:      smtpHandoverGrace,
		},
	}
}

func newConsumerCoordinator(runtimeRepo smtp_domainrepo.RuntimeRepository, projection rebalance.ProjectionSink) *rebalance.Coordinator {
	return &rebalance.Coordinator{
		RuntimeKey:       "smtp:consumer",
		WorkProvider:     consumerWorkProvider{repo: runtimeRepo},
		NodeProvider:     runtimeNodeProvider{repo: runtimeRepo},
		AssignmentSource: consumerAssignmentProvider{repo: runtimeRepo},
		StatusProvider:   consumerStatusProvider{repo: runtimeRepo},
		Projection:       projection,
		Policy:           leaseassign.StickyLeastLoadedPolicy{},
		Config: rebalance.Config{
			AssignmentLeaseTTL: smtpAssignmentLeaseTTL,
			HandoverGrace:      smtpHandoverGrace,
		},
	}
}
