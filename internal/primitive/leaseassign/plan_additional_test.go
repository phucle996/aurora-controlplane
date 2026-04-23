package leaseassign

import (
	"testing"
	"time"
)

func TestPlan_ReusesCurrentOwnerAndMergesMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	workers := []WorkShard{
		{
			WorkID:       "w-1",
			ZoneID:       "z-a",
			GroupKey:     "g-1",
			DesiredState: StateActive,
			Metadata: map[string]string{
				"target_gateway_id": "gw-1",
			},
		},
	}
	nodes := []HealthyNode{
		{NodeID: "dp-a", ZoneID: "z-a", GRPCEndpoint: "grpc://a", Capacity: 1, LeaseExpiresAt: now.Add(20 * time.Second)},
	}
	current := []Assignment{
		{
			WorkID:          "w-1",
			OwnerNodeID:     "dp-a",
			AssignmentState: StateRevoking,
			DesiredState:    StateActive,
			Generation:      99,
			LeaseExpiresAt:  now.Add(10 * time.Second),
			Metadata: map[string]string{
				"keep": "yes",
			},
		},
	}

	out := Plan(workers, nodes, current, now, 30*time.Second, StickyLeastLoadedPolicy{})
	if len(out) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(out))
	}
	if out[0].OwnerNodeID != "dp-a" {
		t.Fatalf("expected reused owner dp-a, got %s", out[0].OwnerNodeID)
	}
	if out[0].AssignmentState != StateActive {
		t.Fatalf("expected assignment state active, got %s", out[0].AssignmentState)
	}
	if out[0].Metadata["keep"] != "yes" {
		t.Fatalf("expected existing metadata preserved")
	}
	if out[0].Metadata["target_gateway_id"] != "gw-1" {
		t.Fatalf("expected work metadata merged")
	}
}

func TestPlan_ReassignsWhenCurrentOwnerCrossZone(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	workers := []WorkShard{
		{WorkID: "w-1", ZoneID: "z-a", DesiredState: StateActive},
	}
	nodes := []HealthyNode{
		{NodeID: "dp-a", ZoneID: "z-a", LeaseExpiresAt: now.Add(20 * time.Second)},
		{NodeID: "dp-b", ZoneID: "z-b", LeaseExpiresAt: now.Add(20 * time.Second)},
	}
	current := []Assignment{
		{
			WorkID:          "w-1",
			OwnerNodeID:     "dp-b",
			AssignmentState: StateActive,
			LeaseExpiresAt:  now.Add(20 * time.Second),
		},
	}

	out := Plan(workers, nodes, current, now, 30*time.Second, StickyLeastLoadedPolicy{})
	if len(out) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(out))
	}
	if out[0].OwnerNodeID != "dp-a" {
		t.Fatalf("expected reassigned to same-zone dp-a, got %s", out[0].OwnerNodeID)
	}
}

func TestPlan_FiltersNonActiveDesiredState(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	workers := []WorkShard{
		{WorkID: "w-1", ZoneID: "z-a", DesiredState: "disabled"},
	}
	nodes := []HealthyNode{
		{NodeID: "dp-a", ZoneID: "z-a", LeaseExpiresAt: now.Add(20 * time.Second)},
	}

	out := Plan(workers, nodes, nil, now, 30*time.Second, StickyLeastLoadedPolicy{})
	if len(out) != 0 {
		t.Fatalf("expected no assignments for non-active desired state, got %d", len(out))
	}
}
