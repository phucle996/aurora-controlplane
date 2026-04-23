package rebalance

import (
	"context"
	"errors"
	"testing"
	"time"

	"controlplane/internal/primitive/leaseassign"
)

func TestCoordinatorReconcile_AppliesTransitionsAndProjectsActive(t *testing.T) {
	t.Parallel()

	work := []leaseassign.WorkShard{
		{WorkID: "w-a-1", ZoneID: "zone-a", DesiredState: leaseassign.StateActive},
		{WorkID: "w-b-1", ZoneID: "zone-b", DesiredState: leaseassign.StateActive},
	}
	nodesByZone := map[string][]leaseassign.HealthyNode{
		"zone-a": {{NodeID: "dp-a1", ZoneID: "zone-a", LeaseExpiresAt: time.Now().Add(time.Minute)}},
		"zone-b": {{NodeID: "dp-b1", ZoneID: "zone-b", LeaseExpiresAt: time.Now().Add(time.Minute)}},
	}
	current := []leaseassign.Assignment{
		{
			WorkID:          "w-a-1",
			OwnerNodeID:     "dp-a1",
			AssignmentState: leaseassign.StateActive,
			DesiredState:    leaseassign.StateActive,
			Generation:      1,
			LeaseExpiresAt:  time.Now().Add(10 * time.Second),
		},
	}

	assign := &fakeAssignmentProvider{rows: current}
	proj := &fakeProjectionSink{}

	c := &Coordinator{
		RuntimeKey:       "smtp:gateway",
		WorkProvider:     fakeWorkProvider{rows: work},
		NodeProvider:     fakeNodeProvider{nodesByZone: nodesByZone},
		AssignmentSource: assign,
		StatusProvider:   fakeStatusProvider{},
		Projection:       proj,
		Config: Config{
			AssignmentLeaseTTL: 30 * time.Second,
			HandoverGrace:      30 * time.Second,
		},
	}

	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if assign.applyCalls != 1 {
		t.Fatalf("expected 1 apply call, got %d", assign.applyCalls)
	}
	if len(assign.applied) != 2 {
		t.Fatalf("expected transitions for 2 works, got %d", len(assign.applied))
	}

	for workID, rows := range assign.applied {
		if len(rows) != 1 {
			t.Fatalf("expected single active row for %s, got %d", workID, len(rows))
		}
		if rows[0].AssignmentState != leaseassign.StateActive {
			t.Fatalf("expected active state for %s", workID)
		}
	}

	if proj.calls != 1 {
		t.Fatalf("expected projection publish once, got %d", proj.calls)
	}
	if proj.runtimeKey != "smtp:gateway" {
		t.Fatalf("unexpected projection runtime key %s", proj.runtimeKey)
	}
	if len(proj.rows) != 2 {
		t.Fatalf("expected 2 projected active rows, got %d", len(proj.rows))
	}
}

func TestCoordinatorReconcile_ErrorPropagation(t *testing.T) {
	t.Parallel()

	c := &Coordinator{
		WorkProvider:     fakeWorkProvider{err: errors.New("boom")},
		NodeProvider:     fakeNodeProvider{},
		AssignmentSource: &fakeAssignmentProvider{},
	}
	if err := c.Reconcile(context.Background()); err == nil {
		t.Fatalf("expected error from work provider")
	}
}

func TestCoordinatorReconcile_NoProvidersNoop(t *testing.T) {
	t.Parallel()

	c := &Coordinator{}
	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatalf("expected nil error on noop reconcile, got %v", err)
	}
}

type fakeWorkProvider struct {
	rows []leaseassign.WorkShard
	err  error
}

func (f fakeWorkProvider) ListWorkShards(_ context.Context) ([]leaseassign.WorkShard, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

type fakeNodeProvider struct {
	nodesByZone map[string][]leaseassign.HealthyNode
	err         error
}

func (f fakeNodeProvider) ListHealthyNodesByZone(_ context.Context, zoneID string, _ time.Time) ([]leaseassign.HealthyNode, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.nodesByZone[zoneID], nil
}

type fakeAssignmentProvider struct {
	rows       []leaseassign.Assignment
	applyErr   error
	applyCalls int
	applied    map[string][]leaseassign.Assignment
}

func (f *fakeAssignmentProvider) ListAssignments(_ context.Context) ([]leaseassign.Assignment, error) {
	return f.rows, nil
}

func (f *fakeAssignmentProvider) ApplyAssignments(_ context.Context, rowsByWork map[string][]leaseassign.Assignment) error {
	f.applyCalls++
	f.applied = rowsByWork
	return f.applyErr
}

type fakeStatusProvider struct {
	status map[string]map[string]RuntimeStatus
	err    error
}

func (f fakeStatusProvider) ListRuntimeStatusByWork(_ context.Context) (map[string]map[string]RuntimeStatus, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.status == nil {
		return map[string]map[string]RuntimeStatus{}, nil
	}
	return f.status, nil
}

type fakeProjectionSink struct {
	calls      int
	runtimeKey string
	rows       []leaseassign.Assignment
	err        error
}

func (f *fakeProjectionSink) PublishActive(_ context.Context, runtimeKey string, rows []leaseassign.Assignment) error {
	f.calls++
	f.runtimeKey = runtimeKey
	f.rows = rows
	return f.err
}
