package leaseassign

import (
	"maps"
	"sort"
	"time"
)

func Plan(
	workers []WorkShard,
	nodes []HealthyNode,
	current []Assignment,
	now time.Time,
	leaseTTL time.Duration,
	policy PlacementPolicy,
) []Assignment {
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}
	if policy == nil {
		policy = StickyLeastLoadedPolicy{}
	}

	activeWork := filterActiveWork(workers)
	healthyNodes := filterHealthyNodes(nodes, now)
	if len(activeWork) == 0 || len(healthyNodes) == 0 {
		return nil
	}

	sort.SliceStable(activeWork, func(i, j int) bool {
		if activeWork[i].GroupKey == activeWork[j].GroupKey {
			return activeWork[i].WorkID < activeWork[j].WorkID
		}
		return activeWork[i].GroupKey < activeWork[j].GroupKey
	})

	loadByNode := make(map[string]int, len(healthyNodes))
	nodeByID := make(map[string]HealthyNode, len(healthyNodes))
	usedByGroup := make(map[string]map[string]struct{})
	for _, node := range healthyNodes {
		loadByNode[node.NodeID] = 0
		nodeByID[node.NodeID] = node
	}

	currentByWork := selectCurrentOwnerByWork(current, now)
	planned := make([]Assignment, 0, len(activeWork))
	generationBase := now.UnixNano()
	generationOffset := int64(0)

	for _, work := range activeWork {
		candidates := filterNodesByZone(healthyNodes, work.ZoneID)
		if len(candidates) == 0 {
			continue
		}
		if _, ok := usedByGroup[work.GroupKey]; !ok {
			usedByGroup[work.GroupKey] = map[string]struct{}{}
		}

		preferredOwner := ""
		if currentRow, ok := currentByWork[work.WorkID]; ok {
			preferredOwner = currentRow.OwnerNodeID
			if currentRow.OwnerNodeID != "" {
				if node, healthy := nodeByID[currentRow.OwnerNodeID]; healthy && node.ZoneID == work.ZoneID {
					assignment := currentRow
					assignment.AssignmentState = StateActive
					assignment.DesiredState = StateActive
					assignment.LeaseExpiresAt = now.Add(leaseTTL)
					assignment.UpdatedAt = now
					assignment.LastTransitionAt = now
					if len(work.Metadata) > 0 {
						if assignment.Metadata == nil {
							assignment.Metadata = map[string]string{}
						}
						maps.Copy(assignment.Metadata, work.Metadata)
					}
					if assignment.CreatedAt.IsZero() {
						assignment.CreatedAt = now
					}
					loadByNode[currentRow.OwnerNodeID] += max(work.Weight, 1)
					if work.GroupKey != "" {
						usedByGroup[work.GroupKey][currentRow.OwnerNodeID] = struct{}{}
					}
					planned = append(planned, assignment)
					continue
				}
			}
		}

		node := policy.ChooseNode(candidates, loadByNode, usedByGroup[work.GroupKey], preferredOwner)
		if node == nil {
			continue
		}

		loadByNode[node.NodeID] += max(work.Weight, 1)
		if work.GroupKey != "" {
			usedByGroup[work.GroupKey][node.NodeID] = struct{}{}
		}
		generationOffset++
		planned = append(planned, Assignment{
			WorkID:           work.WorkID,
			OwnerNodeID:      node.NodeID,
			AssignmentState:  StateActive,
			DesiredState:     StateActive,
			Generation:       generationBase + generationOffset,
			LeaseExpiresAt:   now.Add(leaseTTL),
			LastTransitionAt: now,
			CreatedAt:        now,
			UpdatedAt:        now,
			Metadata: map[string]string{
				"grpc_endpoint": node.GRPCEndpoint,
			},
		})
		if len(work.Metadata) > 0 {
			maps.Copy(planned[len(planned)-1].Metadata, work.Metadata)
		}
	}

	return planned
}

func selectCurrentOwnerByWork(items []Assignment, now time.Time) map[string]Assignment {
	out := make(map[string]Assignment, len(items))
	for _, item := range items {
		if item.WorkID == "" || item.OwnerNodeID == "" || !item.LeaseExpiresAt.After(now) {
			continue
		}
		if existing, ok := out[item.WorkID]; ok {
			if preferredState(item.AssignmentState) > preferredState(existing.AssignmentState) {
				out[item.WorkID] = item
				continue
			}
			if preferredState(item.AssignmentState) == preferredState(existing.AssignmentState) && item.LastTransitionAt.After(existing.LastTransitionAt) {
				out[item.WorkID] = item
			}
			continue
		}
		out[item.WorkID] = item
	}
	return out
}

func preferredState(state string) int {
	switch state {
	case StateActive:
		return 3
	case StatePending:
		return 2
	case StateRevoking:
		return 1
	default:
		return 0
	}
}

func filterActiveWork(items []WorkShard) []WorkShard {
	out := make([]WorkShard, 0, len(items))
	for _, item := range items {
		if item.WorkID == "" {
			continue
		}
		if item.Weight <= 0 {
			item.Weight = 1
		}
		if item.DesiredState == "" {
			item.DesiredState = StateActive
		}
		if item.DesiredState != StateActive {
			continue
		}
		out = append(out, item)
	}
	return out
}

func filterHealthyNodes(items []HealthyNode, now time.Time) []HealthyNode {
	out := make([]HealthyNode, 0, len(items))
	for _, item := range items {
		if item.NodeID == "" || item.ZoneID == "" || !item.LeaseExpiresAt.After(now) {
			continue
		}
		if item.Capacity <= 0 {
			item.Capacity = 1
		}
		out = append(out, item)
	}
	return out
}

func filterNodesByZone(nodes []HealthyNode, zoneID string) []HealthyNode {
	out := make([]HealthyNode, 0, len(nodes))
	for _, node := range nodes {
		if zoneID == "" || node.ZoneID != zoneID {
			continue
		}
		out = append(out, node)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
