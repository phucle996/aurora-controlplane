package rebalance

import (
	"testing"
	"time"

	"controlplane/internal/primitive/leaseassign"
)

func TestComputeTransitions_OwnerChangeCreatesPendingAndRevoking(t *testing.T) {
	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"work-1": {
			{
				WorkID:           "work-1",
				OwnerNodeID:      "dp-old",
				AssignmentState:  leaseassign.StateActive,
				DesiredState:     "active",
				Generation:       10,
				LeaseExpiresAt:   now.Add(20 * time.Second),
				LastTransitionAt: now.Add(-5 * time.Second),
			},
		},
	}
	desired := map[string]string{"work-1": "dp-new"}
	healthy := map[string]struct{}{
		"dp-old": {},
		"dp-new": {},
	}

	out := ComputeTransitions(now, current, desired, nil, healthy, Config{
		AssignmentLeaseTTL: 30 * time.Second,
		HandoverGrace:      30 * time.Second,
	})

	rows := out["work-1"]
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (revoking + pending), got %d", len(rows))
	}

	var hasRevoking bool
	var hasPending bool
	for _, row := range rows {
		if row.OwnerNodeID == "dp-old" && row.AssignmentState == leaseassign.StateRevoking {
			hasRevoking = true
		}
		if row.OwnerNodeID == "dp-new" && row.AssignmentState == leaseassign.StatePending {
			hasPending = true
		}
	}
	if !hasRevoking || !hasPending {
		t.Fatalf("expected revoking(old) and pending(new), got %#v", rows)
	}
}

func TestComputeTransitions_PromotesPendingWhenRevokingDone(t *testing.T) {
	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"work-1": {
			{
				WorkID:           "work-1",
				OwnerNodeID:      "dp-old",
				AssignmentState:  leaseassign.StateRevoking,
				DesiredState:     "active",
				Generation:       10,
				LeaseExpiresAt:   now.Add(20 * time.Second),
				LastTransitionAt: now.Add(-20 * time.Second),
			},
			{
				WorkID:           "work-1",
				OwnerNodeID:      "dp-new",
				AssignmentState:  leaseassign.StatePending,
				DesiredState:     "active",
				Generation:       10,
				LeaseExpiresAt:   now.Add(20 * time.Second),
				LastTransitionAt: now.Add(-20 * time.Second),
			},
		},
	}
	desired := map[string]string{"work-1": "dp-new"}
	statuses := map[string]map[string]RuntimeStatus{
		"work-1": {
			"dp-old": {
				WorkID:          "work-1",
				OwnerNodeID:     "dp-old",
				AssignmentState: leaseassign.StateRevoking,
				RevokingDone:    true,
			},
		},
	}

	out := ComputeTransitions(now, current, desired, statuses, map[string]struct{}{"dp-new": {}, "dp-old": {}}, Config{
		AssignmentLeaseTTL: 30 * time.Second,
		HandoverGrace:      30 * time.Second,
	})
	rows := out["work-1"]
	if len(rows) != 1 {
		t.Fatalf("expected single active row after promote, got %d", len(rows))
	}
	if rows[0].OwnerNodeID != "dp-new" || rows[0].AssignmentState != leaseassign.StateActive {
		t.Fatalf("expected active on dp-new after promote, got %#v", rows[0])
	}
}

func TestComputeTransitions_PromotesPendingWhenOldNodeDead(t *testing.T) {
	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"work-1": {
			{
				WorkID:           "work-1",
				OwnerNodeID:      "dp-old",
				AssignmentState:  leaseassign.StateRevoking,
				DesiredState:     "active",
				Generation:       10,
				LeaseExpiresAt:   now.Add(20 * time.Second),
				LastTransitionAt: now.Add(-40 * time.Second),
			},
			{
				WorkID:           "work-1",
				OwnerNodeID:      "dp-new",
				AssignmentState:  leaseassign.StatePending,
				DesiredState:     "active",
				Generation:       10,
				LeaseExpiresAt:   now.Add(20 * time.Second),
				LastTransitionAt: now.Add(-40 * time.Second),
			},
		},
	}
	desired := map[string]string{"work-1": "dp-new"}
	healthy := map[string]struct{}{"dp-new": {}}

	out := ComputeTransitions(now, current, desired, nil, healthy, Config{
		AssignmentLeaseTTL: 30 * time.Second,
		HandoverGrace:      30 * time.Second,
	})
	rows := out["work-1"]
	if len(rows) != 1 {
		t.Fatalf("expected single active row after dead owner promote, got %d", len(rows))
	}
	if rows[0].OwnerNodeID != "dp-new" || rows[0].AssignmentState != leaseassign.StateActive {
		t.Fatalf("expected active on dp-new after old node death, got %#v", rows[0])
	}
}
