package leaseassign

import "controlplane/internal/primitive/clusterload"

type PlacementPolicy interface {
	ChooseNode(nodes []HealthyNode, loadByNode map[string]int, usedInGroup map[string]struct{}, preferredOwner string) *HealthyNode
}

type StickyLeastLoadedPolicy struct{}

func (StickyLeastLoadedPolicy) ChooseNode(nodes []HealthyNode, loadByNode map[string]int, usedInGroup map[string]struct{}, preferredOwner string) *HealthyNode {
	candidates := filterUnusedNodes(nodes, usedInGroup)
	if len(candidates) == 0 {
		candidates = nodes
	}
	if len(candidates) == 0 {
		return nil
	}

	best := candidates[0]
	bestScore := clusterload.NormalizedScore(loadByNode[best.NodeID], best.Capacity)
	for _, node := range candidates[1:] {
		score := clusterload.NormalizedScore(loadByNode[node.NodeID], node.Capacity)
		if score < bestScore ||
			(score == bestScore && node.NodeID == preferredOwner) ||
			(score == bestScore && best.NodeID != preferredOwner && node.NodeID < best.NodeID) {
			best = node
			bestScore = score
		}
	}
	copy := best
	return &copy
}

func filterUnusedNodes(nodes []HealthyNode, usedInGroup map[string]struct{}) []HealthyNode {
	if len(usedInGroup) == 0 {
		return nodes
	}
	out := make([]HealthyNode, 0, len(nodes))
	for _, node := range nodes {
		if _, used := usedInGroup[node.NodeID]; used {
			continue
		}
		out = append(out, node)
	}
	return out
}
