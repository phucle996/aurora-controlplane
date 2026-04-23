package rebalance

import (
	"sort"
	"time"

	"controlplane/internal/primitive/leaseassign"
)

func ComputeTransitions(
	now time.Time,
	current map[string][]leaseassign.Assignment,
	desiredOwnerByWork map[string]string,
	statusByWork map[string]map[string]RuntimeStatus,
	healthyNodes map[string]struct{},
	cfg Config,
) map[string][]leaseassign.Assignment {
	cfg = defaultConfig(cfg)

	workIDs := collectWorkIDs(current, desiredOwnerByWork)
	out := make(map[string][]leaseassign.Assignment, len(workIDs))

	for _, workID := range workIDs {
		desiredOwner := desiredOwnerByWork[workID]
		rows := cloneRows(current[workID])
		if desiredOwner == "" {
			out[workID] = nil
			continue
		}

		active := findRow(rows, leaseassign.StateActive)
		pending := findRowByOwnerAndState(rows, desiredOwner, leaseassign.StatePending)
		revokingRows := findRows(rows, leaseassign.StateRevoking)
		nextGen := nextGeneration(rows, now.UnixNano())

		switch {
		case active != nil && active.OwnerNodeID == desiredOwner:
			row := refreshRow(*active, now, cfg.AssignmentLeaseTTL)
			row.AssignmentState = leaseassign.StateActive
			row.DesiredState = leaseassign.StateActive
			out[workID] = []leaseassign.Assignment{row}
			continue

		case active != nil && active.OwnerNodeID != desiredOwner:
			revoking := refreshRow(*active, now, cfg.AssignmentLeaseTTL)
			revoking.AssignmentState = leaseassign.StateRevoking
			revoking.DesiredState = leaseassign.StateActive

			next := pending
			if next == nil {
				newPending := newPendingAssignment(now, workID, desiredOwner, nextGen, cfg.AssignmentLeaseTTL)
				out[workID] = []leaseassign.Assignment{revoking, newPending}
				continue
			}
			refreshedPending := refreshRow(*next, now, cfg.AssignmentLeaseTTL)
			refreshedPending.AssignmentState = leaseassign.StatePending
			refreshedPending.DesiredState = leaseassign.StateActive
			if refreshedPending.Generation < nextGen {
				refreshedPending.Generation = nextGen
			}
			out[workID] = []leaseassign.Assignment{revoking, refreshedPending}
			continue
		}

		if pending == nil {
			pendingCandidate := findRow(rows, leaseassign.StatePending)
			if pendingCandidate != nil && pendingCandidate.OwnerNodeID == desiredOwner {
				pending = pendingCandidate
			}
		}
		if pending == nil {
			pendingRow := newPendingAssignment(now, workID, desiredOwner, nextGen, cfg.AssignmentLeaseTTL)
			pending = &pendingRow
		}

		if len(revokingRows) == 0 {
			activeRow := refreshRow(*pending, now, cfg.AssignmentLeaseTTL)
			activeRow.AssignmentState = leaseassign.StateActive
			activeRow.DesiredState = leaseassign.StateActive
			if activeRow.Generation < nextGen {
				activeRow.Generation = nextGen
			}
			out[workID] = []leaseassign.Assignment{activeRow}
			continue
		}

		if canPromotePending(now, workID, revokingRows, statusByWork[workID], healthyNodes, cfg.HandoverGrace) {
			activeRow := refreshRow(*pending, now, cfg.AssignmentLeaseTTL)
			activeRow.AssignmentState = leaseassign.StateActive
			activeRow.DesiredState = leaseassign.StateActive
			if activeRow.Generation < nextGen {
				activeRow.Generation = nextGen
			}
			out[workID] = []leaseassign.Assignment{activeRow}
			continue
		}

		kept := make([]leaseassign.Assignment, 0, len(revokingRows)+1)
		for _, row := range revokingRows {
			kept = append(kept, refreshRow(row, now, cfg.AssignmentLeaseTTL))
		}
		refreshedPending := refreshRow(*pending, now, cfg.AssignmentLeaseTTL)
		refreshedPending.AssignmentState = leaseassign.StatePending
		refreshedPending.DesiredState = leaseassign.StateActive
		if refreshedPending.Generation < nextGen {
			refreshedPending.Generation = nextGen
		}
		kept = append(kept, refreshedPending)
		out[workID] = kept
	}

	return out
}

func canPromotePending(
	now time.Time,
	workID string,
	revokingRows []leaseassign.Assignment,
	statusByOwner map[string]RuntimeStatus,
	healthyNodes map[string]struct{},
	grace time.Duration,
) bool {
	for _, row := range revokingRows {
		if statusByOwner != nil {
			if status, ok := statusByOwner[row.OwnerNodeID]; ok && status.WorkID == workID && status.RevokingDone {
				continue
			}
		}
		if _, healthy := healthyNodes[row.OwnerNodeID]; !healthy {
			continue
		}
		if now.Sub(row.LastTransitionAt) >= grace {
			continue
		}
		return false
	}
	return true
}

func refreshRow(row leaseassign.Assignment, now time.Time, ttl time.Duration) leaseassign.Assignment {
	refreshed := row
	refreshed.LeaseExpiresAt = now.Add(ttl)
	refreshed.UpdatedAt = now
	if refreshed.LastTransitionAt.IsZero() {
		refreshed.LastTransitionAt = now
	}
	if refreshed.CreatedAt.IsZero() {
		refreshed.CreatedAt = now
	}
	return refreshed
}

func collectWorkIDs(current map[string][]leaseassign.Assignment, desired map[string]string) []string {
	set := make(map[string]struct{}, len(current)+len(desired))
	for workID := range current {
		if workID != "" {
			set[workID] = struct{}{}
		}
	}
	for workID := range desired {
		if workID != "" {
			set[workID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func cloneRows(rows []leaseassign.Assignment) []leaseassign.Assignment {
	if len(rows) == 0 {
		return nil
	}
	out := make([]leaseassign.Assignment, len(rows))
	copy(out, rows)
	return out
}

func findRow(rows []leaseassign.Assignment, state string) *leaseassign.Assignment {
	for i := range rows {
		if rows[i].AssignmentState == state {
			return &rows[i]
		}
	}
	return nil
}

func findRowByOwnerAndState(rows []leaseassign.Assignment, owner, state string) *leaseassign.Assignment {
	for i := range rows {
		if rows[i].OwnerNodeID == owner && rows[i].AssignmentState == state {
			return &rows[i]
		}
	}
	return nil
}

func findRows(rows []leaseassign.Assignment, state string) []leaseassign.Assignment {
	out := make([]leaseassign.Assignment, 0)
	for _, row := range rows {
		if row.AssignmentState == state {
			out = append(out, row)
		}
	}
	return out
}

func nextGeneration(rows []leaseassign.Assignment, fallback int64) int64 {
	maxGen := fallback
	for _, row := range rows {
		if row.Generation > maxGen {
			maxGen = row.Generation
		}
	}
	return maxGen + 1
}
