package rebalance

import (
	"context"
	"time"

	"controlplane/internal/primitive/leaseassign"
)

type WorkProvider interface {
	ListWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error)
}

type NodeProvider interface {
	ListHealthyNodesByZone(ctx context.Context, zoneID string, now time.Time) ([]leaseassign.HealthyNode, error)
}

type AssignmentProvider interface {
	ListAssignments(ctx context.Context) ([]leaseassign.Assignment, error)
	ApplyAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error
}

type RuntimeStatusProvider interface {
	ListRuntimeStatusByWork(ctx context.Context) (map[string]map[string]RuntimeStatus, error)
}

type ProjectionSink interface {
	PublishActive(ctx context.Context, runtimeKey string, rows []leaseassign.Assignment) error
}

type Coordinator struct {
	RuntimeKey       string
	WorkProvider     WorkProvider
	NodeProvider     NodeProvider
	AssignmentSource AssignmentProvider
	StatusProvider   RuntimeStatusProvider
	Projection       ProjectionSink
	Policy           leaseassign.PlacementPolicy
	Config           Config
}

func (c *Coordinator) Reconcile(ctx context.Context) error {
	if c == nil || c.WorkProvider == nil || c.NodeProvider == nil || c.AssignmentSource == nil {
		return nil
	}
	now := time.Now().UTC()

	workShards, err := c.WorkProvider.ListWorkShards(ctx)
	if err != nil {
		return err
	}
	currentRows, err := c.AssignmentSource.ListAssignments(ctx)
	if err != nil {
		return err
	}

	statusByWork := map[string]map[string]RuntimeStatus{}
	if c.StatusProvider != nil {
		statusByWork, err = c.StatusProvider.ListRuntimeStatusByWork(ctx)
		if err != nil {
			return err
		}
	}

	workByZone := groupWorkByZone(workShards)
	currentByWork := groupAssignmentsByWork(currentRows)

	desiredOwnerByWork := make(map[string]string, len(workShards))
	healthyNodeSet := map[string]struct{}{}

	for zoneID, zoneWork := range workByZone {
		nodes, err := c.NodeProvider.ListHealthyNodesByZone(ctx, zoneID, now)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			healthyNodeSet[node.NodeID] = struct{}{}
		}

		zoneCurrent := collectCurrentRowsForWorkSet(currentRows, zoneWork)
		planned := leaseassign.Plan(zoneWork, nodes, zoneCurrent, now, c.Config.AssignmentLeaseTTL, c.Policy)
		for _, row := range planned {
			if row.WorkID == "" {
				continue
			}
			desiredOwnerByWork[row.WorkID] = row.OwnerNodeID
		}
	}

	transitions := ComputeTransitions(now, currentByWork, desiredOwnerByWork, statusByWork, healthyNodeSet, c.Config)
	if err := c.AssignmentSource.ApplyAssignments(ctx, transitions); err != nil {
		return err
	}

	if c.Projection != nil {
		activeRows := flattenActiveRows(transitions)
		if err := c.Projection.PublishActive(ctx, c.RuntimeKey, activeRows); err != nil {
			return err
		}
	}

	return nil
}

func groupWorkByZone(items []leaseassign.WorkShard) map[string][]leaseassign.WorkShard {
	out := make(map[string][]leaseassign.WorkShard)
	for _, item := range items {
		if item.WorkID == "" || item.ZoneID == "" {
			continue
		}
		out[item.ZoneID] = append(out[item.ZoneID], item)
	}
	return out
}

func groupAssignmentsByWork(items []leaseassign.Assignment) map[string][]leaseassign.Assignment {
	out := make(map[string][]leaseassign.Assignment)
	for _, item := range items {
		if item.WorkID == "" {
			continue
		}
		out[item.WorkID] = append(out[item.WorkID], item)
	}
	return out
}

func collectCurrentRowsForWorkSet(rows []leaseassign.Assignment, works []leaseassign.WorkShard) []leaseassign.Assignment {
	workSet := make(map[string]struct{}, len(works))
	for _, work := range works {
		workSet[work.WorkID] = struct{}{}
	}
	out := make([]leaseassign.Assignment, 0, len(rows))
	for _, row := range rows {
		if _, ok := workSet[row.WorkID]; !ok {
			continue
		}
		out = append(out, row)
	}
	return out
}

func flattenActiveRows(rowsByWork map[string][]leaseassign.Assignment) []leaseassign.Assignment {
	out := make([]leaseassign.Assignment, 0)
	for _, rows := range rowsByWork {
		for _, row := range rows {
			if row.AssignmentState == leaseassign.StateActive {
				out = append(out, row)
			}
		}
	}
	return out
}
