package rebalance

import (
	"testing"
	"time"

	"controlplane/internal/primitive/leaseassign"
)

func TestComputeTransitions_KeepPendingWhenRevokingNotDoneAndOwnerHealthy(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"w-1": {
			{
				WorkID:           "w-1",
				OwnerNodeID:      "dp-old",
				AssignmentState:  leaseassign.StateRevoking,
				DesiredState:     leaseassign.StateActive,
				Generation:       10,
				LeaseExpiresAt:   now.Add(10 * time.Second),
				LastTransitionAt: now.Add(-10 * time.Second),
			},
			{
				WorkID:           "w-1",
				OwnerNodeID:      "dp-new",
				AssignmentState:  leaseassign.StatePending,
				DesiredState:     leaseassign.StateActive,
				Generation:       10,
				LeaseExpiresAt:   now.Add(10 * time.Second),
				LastTransitionAt: now.Add(-10 * time.Second),
			},
		},
	}
	desired := map[string]string{"w-1": "dp-new"}
	healthy := map[string]struct{}{
		"dp-old": {},
		"dp-new": {},
	}

	out := ComputeTransitions(now, current, desired, nil, healthy, Config{
		AssignmentLeaseTTL: 30 * time.Second,
		HandoverGrace:      30 * time.Second,
	})
	rows := out["w-1"]
	if len(rows) != 2 {
		t.Fatalf("expected pending+revoking to remain, got %d rows", len(rows))
	}
}

func TestComputeTransitions_PromotesPendingWhenGraceTimeout(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"w-1": {
			{
				WorkID:           "w-1",
				OwnerNodeID:      "dp-old",
				AssignmentState:  leaseassign.StateRevoking,
				DesiredState:     leaseassign.StateActive,
				Generation:       10,
				LeaseExpiresAt:   now.Add(10 * time.Second),
				LastTransitionAt: now.Add(-40 * time.Second),
			},
			{
				WorkID:           "w-1",
				OwnerNodeID:      "dp-new",
				AssignmentState:  leaseassign.StatePending,
				DesiredState:     leaseassign.StateActive,
				Generation:       10,
				LeaseExpiresAt:   now.Add(10 * time.Second),
				LastTransitionAt: now.Add(-40 * time.Second),
			},
		},
	}
	desired := map[string]string{"w-1": "dp-new"}
	healthy := map[string]struct{}{
		"dp-old": {},
		"dp-new": {},
	}

	out := ComputeTransitions(now, current, desired, nil, healthy, Config{
		AssignmentLeaseTTL: 30 * time.Second,
		HandoverGrace:      30 * time.Second,
	})
	rows := out["w-1"]
	if len(rows) != 1 {
		t.Fatalf("expected single active row after grace timeout, got %d", len(rows))
	}
	if rows[0].OwnerNodeID != "dp-new" || rows[0].AssignmentState != leaseassign.StateActive {
		t.Fatalf("expected dp-new active, got %+v", rows[0])
	}
}

func TestComputeTransitions_ClearsWhenNoDesiredOwner(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"w-1": {
			{
				WorkID:          "w-1",
				OwnerNodeID:     "dp-old",
				AssignmentState: leaseassign.StateActive,
				LeaseExpiresAt:  now.Add(20 * time.Second),
			},
		},
	}
	desired := map[string]string{}

	out := ComputeTransitions(now, current, desired, nil, nil, Config{})
	rows, ok := out["w-1"]
	if !ok {
		t.Fatalf("expected work key in output")
	}
	if len(rows) != 0 {
		t.Fatalf("expected cleared rows when no desired owner, got %d", len(rows))
	}
}

func TestComputeTransitions_RefreshesActiveWhenDesiredOwnerUnchanged(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	current := map[string][]leaseassign.Assignment{
		"w-1": {
			{
				WorkID:          "w-1",
				OwnerNodeID:     "dp-a",
				AssignmentState: leaseassign.StateActive,
				DesiredState:    leaseassign.StateActive,
				Generation:      7,
				LeaseExpiresAt:  now.Add(5 * time.Second),
			},
		},
	}
	desired := map[string]string{"w-1": "dp-a"}

	out := ComputeTransitions(now, current, desired, nil, map[string]struct{}{"dp-a": {}}, Config{
		AssignmentLeaseTTL: 1 * time.Minute,
		HandoverGrace:      1 * time.Minute,
	})
	rows := out["w-1"]
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].OwnerNodeID != "dp-a" || rows[0].AssignmentState != leaseassign.StateActive {
		t.Fatalf("expected active on same owner, got %+v", rows[0])
	}
	if !rows[0].LeaseExpiresAt.After(current["w-1"][0].LeaseExpiresAt) {
		t.Fatalf("expected lease refresh")
	}
}
