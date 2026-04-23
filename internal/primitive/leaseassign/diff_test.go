package leaseassign

import "testing"

func TestPublishedChanges(t *testing.T) {
	t.Parallel()

	current := []Assignment{
		{
			WorkID:          "w-1",
			OwnerNodeID:     "dp-a",
			AssignmentState: StateActive,
			DesiredState:    StateActive,
			Generation:      10,
			Metadata: map[string]string{
				"k": "v",
			},
		},
	}

	desiredSame := []Assignment{
		{
			WorkID:          "w-1",
			OwnerNodeID:     "dp-a",
			AssignmentState: StateActive,
			DesiredState:    StateActive,
			Generation:      10,
			Metadata: map[string]string{
				"k": "v",
			},
		},
	}

	changes := PublishedChanges(current, desiredSame)
	if len(changes) != 0 {
		t.Fatalf("expected no changes, got %d", len(changes))
	}

	desiredChanged := []Assignment{
		{
			WorkID:          "w-1",
			OwnerNodeID:     "dp-b",
			AssignmentState: StatePending,
			DesiredState:    StateActive,
			Generation:      11,
			Metadata: map[string]string{
				"k": "next",
			},
		},
		{
			WorkID:          "w-2",
			OwnerNodeID:     "dp-a",
			AssignmentState: StateActive,
			DesiredState:    StateActive,
			Generation:      12,
		},
	}

	changes = PublishedChanges(current, desiredChanged)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].WorkID != "w-1" && changes[1].WorkID != "w-1" {
		t.Fatalf("expected changed work w-1 in changes")
	}
}
