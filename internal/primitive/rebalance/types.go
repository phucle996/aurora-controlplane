package rebalance

import (
	"time"

	"controlplane/internal/primitive/leaseassign"
)

type Config struct {
	AssignmentLeaseTTL time.Duration
	HandoverGrace      time.Duration
}

type RuntimeStatus struct {
	WorkID          string
	OwnerNodeID     string
	AssignmentState string
	RevokingDone    bool
	Generation      int64
}

func defaultConfig(cfg Config) Config {
	if cfg.AssignmentLeaseTTL <= 0 {
		cfg.AssignmentLeaseTTL = 30 * time.Second
	}
	if cfg.HandoverGrace <= 0 {
		cfg.HandoverGrace = cfg.AssignmentLeaseTTL
	}
	return cfg
}

func newActiveAssignment(now time.Time, workID, nodeID string, generation int64, leaseTTL time.Duration) leaseassign.Assignment {
	return leaseassign.Assignment{
		WorkID:           workID,
		OwnerNodeID:      nodeID,
		AssignmentState:  leaseassign.StateActive,
		DesiredState:     leaseassign.StateActive,
		Generation:       generation,
		LeaseExpiresAt:   now.Add(leaseTTL),
		LastTransitionAt: now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func newPendingAssignment(now time.Time, workID, nodeID string, generation int64, leaseTTL time.Duration) leaseassign.Assignment {
	return leaseassign.Assignment{
		WorkID:           workID,
		OwnerNodeID:      nodeID,
		AssignmentState:  leaseassign.StatePending,
		DesiredState:     leaseassign.StateActive,
		Generation:       generation,
		LeaseExpiresAt:   now.Add(leaseTTL),
		LastTransitionAt: now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
