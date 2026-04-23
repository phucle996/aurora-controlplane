package leaseassign

import "maps"

func PublishedChanges(current []Assignment, desired []Assignment) []PublishedAssignment {
	currentByWork := make(map[string]Assignment, len(current))
	for _, row := range current {
		if row.WorkID == "" {
			continue
		}
		currentByWork[row.WorkID] = row
	}

	out := make([]PublishedAssignment, 0)
	for _, row := range desired {
		if row.WorkID == "" {
			continue
		}
		old, ok := currentByWork[row.WorkID]
		if ok && samePublished(old, row) {
			continue
		}
		out = append(out, PublishedAssignment{
			WorkID:           row.WorkID,
			OwnerNodeID:      row.OwnerNodeID,
			AssignmentState:  row.AssignmentState,
			DesiredState:     row.DesiredState,
			Generation:       row.Generation,
			LeaseExpiresAt:   row.LeaseExpiresAt,
			LastTransitionAt: row.LastTransitionAt,
			Metadata:         maps.Clone(row.Metadata),
		})
	}
	return out
}

func samePublished(a Assignment, b Assignment) bool {
	return a.WorkID == b.WorkID &&
		a.OwnerNodeID == b.OwnerNodeID &&
		a.AssignmentState == b.AssignmentState &&
		a.DesiredState == b.DesiredState &&
		a.Generation == b.Generation &&
		maps.Equal(a.Metadata, b.Metadata)
}
