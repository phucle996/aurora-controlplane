package leaseassign

import (
	"testing"
	"time"
)

func TestPlan_PrefersLeastLoadedByCapacity(t *testing.T) {
	now := time.Now().UTC()

	workers := []WorkShard{
		{WorkID: "w-1", ZoneID: "z-a", GroupKey: "g-1", Weight: 1, DesiredState: "active"},
		{WorkID: "w-2", ZoneID: "z-a", GroupKey: "g-1", Weight: 1, DesiredState: "active"},
		{WorkID: "w-3", ZoneID: "z-a", GroupKey: "g-2", Weight: 1, DesiredState: "active"},
	}
	nodes := []HealthyNode{
		{NodeID: "dp-a", ZoneID: "z-a", Capacity: 1, LeaseExpiresAt: now.Add(30 * time.Second)},
		{NodeID: "dp-b", ZoneID: "z-a", Capacity: 4, LeaseExpiresAt: now.Add(30 * time.Second)},
	}
	current := []Assignment{
		{WorkID: "old", OwnerNodeID: "dp-a", AssignmentState: StateActive, LeaseExpiresAt: now.Add(30 * time.Second)},
	}

	planned := Plan(workers, nodes, current, now, 30*time.Second, StickyLeastLoadedPolicy{})
	if len(planned) != 3 {
		t.Fatalf("expected 3 planned assignments, got %d", len(planned))
	}

	countByNode := map[string]int{}
	for _, item := range planned {
		countByNode[item.OwnerNodeID]++
	}
	if countByNode["dp-b"] <= countByNode["dp-a"] {
		t.Fatalf("expected dp-b to receive more work due to capacity, got dp-a=%d dp-b=%d", countByNode["dp-a"], countByNode["dp-b"])
	}
}

func TestPlan_GroupSpread_WhenPossible(t *testing.T) {
	now := time.Now().UTC()

	workers := []WorkShard{
		{WorkID: "w-1", ZoneID: "z-a", GroupKey: "same-group", Weight: 1, DesiredState: "active"},
		{WorkID: "w-2", ZoneID: "z-a", GroupKey: "same-group", Weight: 1, DesiredState: "active"},
	}
	nodes := []HealthyNode{
		{NodeID: "dp-a", ZoneID: "z-a", Capacity: 1, LeaseExpiresAt: now.Add(30 * time.Second)},
		{NodeID: "dp-b", ZoneID: "z-a", Capacity: 1, LeaseExpiresAt: now.Add(30 * time.Second)},
	}

	planned := Plan(workers, nodes, nil, now, 30*time.Second, StickyLeastLoadedPolicy{})
	if len(planned) != 2 {
		t.Fatalf("expected 2 planned assignments, got %d", len(planned))
	}
	if planned[0].OwnerNodeID == planned[1].OwnerNodeID {
		t.Fatalf("expected same group to spread across nodes when possible, got owner=%s", planned[0].OwnerNodeID)
	}
}

func TestPlan_SameZoneOnly(t *testing.T) {
	now := time.Now().UTC()
	workers := []WorkShard{
		{WorkID: "w-1", ZoneID: "z-a", Weight: 1, DesiredState: "active"},
	}
	nodes := []HealthyNode{
		{NodeID: "dp-a", ZoneID: "z-b", Capacity: 10, LeaseExpiresAt: now.Add(30 * time.Second)},
	}

	planned := Plan(workers, nodes, nil, now, 30*time.Second, StickyLeastLoadedPolicy{})
	if len(planned) != 0 {
		t.Fatalf("expected no assignment for cross-zone only node, got %d", len(planned))
	}
}
