package repository

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"controlplane/internal/primitive/leaseassign"
	"controlplane/internal/primitive/rebalance"

	"github.com/jackc/pgx/v5"
)

func (r *RuntimeRepository) ListGatewayWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT gs.gateway_id, gs.shard_id, g.zone_id
		FROM smtp.gateway_shards gs
		JOIN smtp.gateways g ON g.id = gs.gateway_id
		WHERE g.status = 'active'
		  AND gs.desired_state = 'active'
		  AND g.zone_id IS NOT NULL
		ORDER BY gs.gateway_id, gs.shard_id
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway work shards: %w", err)
	}
	defer rows.Close()

	out := make([]leaseassign.WorkShard, 0)
	for rows.Next() {
		var gatewayID string
		var shardID int
		var zoneID string
		if err := rows.Scan(&gatewayID, &shardID, &zoneID); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway work shard: %w", err)
		}
		out = append(out, leaseassign.WorkShard{
			WorkID:       gatewayWorkID(gatewayID, shardID),
			ZoneID:       zoneID,
			GroupKey:     "gateway:" + gatewayID,
			Weight:       1,
			DesiredState: leaseassign.StateActive,
			Metadata: map[string]string{
				"gateway_id": gatewayID,
				"shard_id":   strconv.Itoa(shardID),
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateway work shards: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ListConsumerWorkShards(ctx context.Context) ([]leaseassign.WorkShard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cs.consumer_id, cs.shard_id, c.zone_id
		FROM smtp.consumer_shards cs
		JOIN smtp.consumers c ON c.id = cs.consumer_id
		WHERE c.status = 'active'
		  AND cs.desired_state = 'active'
		  AND c.zone_id IS NOT NULL
		ORDER BY cs.consumer_id, cs.shard_id
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumer work shards: %w", err)
	}
	defer rows.Close()

	compatibleTargets, err := r.compatibleGatewayTargetsByConsumer(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]leaseassign.WorkShard, 0)
	for rows.Next() {
		var consumerID string
		var shardID int
		var zoneID string
		if err := rows.Scan(&consumerID, &shardID, &zoneID); err != nil {
			return nil, fmt.Errorf("smtp repo: scan consumer work shard: %w", err)
		}

		targets := compatibleTargets[consumerID]
		if len(targets) == 0 {
			continue
		}
		target := chooseGatewayTargetByShard(targets, shardID)
		if target == nil {
			continue
		}

		out = append(out, leaseassign.WorkShard{
			WorkID:       consumerWorkID(consumerID, shardID),
			ZoneID:       zoneID,
			GroupKey:     "consumer:" + consumerID,
			Weight:       1,
			DesiredState: leaseassign.StateActive,
			Metadata: map[string]string{
				"consumer_id":                      consumerID,
				"shard_id":                         strconv.Itoa(shardID),
				"target_gateway_id":                target.GatewayID,
				"target_gateway_shard_id":          strconv.Itoa(target.ShardID),
				"target_gateway_data_plane_id":     target.DataPlaneID,
				"target_gateway_grpc_endpoint":     target.GRPCEndpoint,
				"target_gateway_assignment_zoneID": target.ZoneID,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate consumer work shards: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ListGatewayAssignmentsForReconcile(ctx context.Context) ([]leaseassign.Assignment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT gateway_id, shard_id, data_plane_id, generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at
		FROM smtp.gateway_shard_assignments
		ORDER BY gateway_id, shard_id, updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway assignments for reconcile: %w", err)
	}
	defer rows.Close()

	out := make([]leaseassign.Assignment, 0)
	for rows.Next() {
		var gatewayID string
		var shardID int
		var row leaseassign.Assignment
		if err := rows.Scan(&gatewayID, &shardID, &row.OwnerNodeID, &row.Generation, &row.AssignmentState, &row.DesiredState, &row.LeaseExpiresAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway assignment for reconcile: %w", err)
		}
		row.WorkID = gatewayWorkID(gatewayID, shardID)
		row.LastTransitionAt = row.UpdatedAt
		row.Metadata = map[string]string{
			"gateway_id": gatewayID,
			"shard_id":   strconv.Itoa(shardID),
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateway assignments for reconcile: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ListConsumerAssignmentsForReconcile(ctx context.Context) ([]leaseassign.Assignment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT consumer_id, shard_id, data_plane_id, generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at,
		       target_gateway_id, target_gateway_shard_id, target_gateway_data_plane_id, target_gateway_grpc_endpoint
		FROM smtp.consumer_assignments
		ORDER BY consumer_id, shard_id, updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumer assignments for reconcile: %w", err)
	}
	defer rows.Close()

	out := make([]leaseassign.Assignment, 0)
	for rows.Next() {
		var consumerID string
		var shardID int
		var targetGatewayID *string
		var targetGatewayShardID *int
		var targetGatewayDataPlaneID *string
		var targetGatewayGRPCEndpoint string
		var row leaseassign.Assignment
		if err := rows.Scan(
			&consumerID,
			&shardID,
			&row.OwnerNodeID,
			&row.Generation,
			&row.AssignmentState,
			&row.DesiredState,
			&row.LeaseExpiresAt,
			&row.CreatedAt,
			&row.UpdatedAt,
			&targetGatewayID,
			&targetGatewayShardID,
			&targetGatewayDataPlaneID,
			&targetGatewayGRPCEndpoint,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan consumer assignment for reconcile: %w", err)
		}
		row.WorkID = consumerWorkID(consumerID, shardID)
		row.LastTransitionAt = row.UpdatedAt
		row.Metadata = map[string]string{
			"consumer_id":                  consumerID,
			"shard_id":                     strconv.Itoa(shardID),
			"target_gateway_grpc_endpoint": targetGatewayGRPCEndpoint,
		}
		if targetGatewayID != nil {
			row.Metadata["target_gateway_id"] = *targetGatewayID
		}
		if targetGatewayShardID != nil {
			row.Metadata["target_gateway_shard_id"] = strconv.Itoa(*targetGatewayShardID)
		}
		if targetGatewayDataPlaneID != nil {
			row.Metadata["target_gateway_data_plane_id"] = *targetGatewayDataPlaneID
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate consumer assignments for reconcile: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ListHealthyRuntimeNodesByZone(ctx context.Context, zoneID string, now time.Time) ([]leaseassign.HealthyNode, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dp.id, dp.zone_id, COALESCE(dp.grpc_endpoint, ''), COALESCE(h.capacity, 1)
		FROM core.data_planes dp
		LEFT JOIN smtp.runtime_heartbeats h ON h.data_plane_id = dp.id
		WHERE dp.zone_id = $1
		  AND dp.status = 'healthy'
		ORDER BY COALESCE(h.capacity, 1) DESC, dp.last_seen_at DESC NULLS LAST, dp.created_at ASC
	`, zoneID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list healthy runtime nodes by zone: %w", err)
	}
	defer rows.Close()

	out := make([]leaseassign.HealthyNode, 0)
	for rows.Next() {
		var node leaseassign.HealthyNode
		if err := rows.Scan(&node.NodeID, &node.ZoneID, &node.GRPCEndpoint, &node.Capacity); err != nil {
			return nil, fmt.Errorf("smtp repo: scan healthy runtime node: %w", err)
		}
		node.LeaseExpiresAt = now.Add(30 * time.Second)
		out = append(out, node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate healthy runtime nodes: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ListGatewayRuntimeStatusByWork(ctx context.Context) (map[string]map[string]rebalance.RuntimeStatus, error) {
	rows, err := r.db.Query(ctx, `
		SELECT gateway_id, shard_id, data_plane_id, assignment_state, revoking_done, generation
		FROM smtp.gateway_runtime_statuses
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway runtime statuses by work: %w", err)
	}
	defer rows.Close()

	out := map[string]map[string]rebalance.RuntimeStatus{}
	for rows.Next() {
		var gatewayID string
		var shardID int
		var status rebalance.RuntimeStatus
		if err := rows.Scan(&gatewayID, &shardID, &status.OwnerNodeID, &status.AssignmentState, &status.RevokingDone, &status.Generation); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway runtime status by work: %w", err)
		}
		status.WorkID = gatewayWorkID(gatewayID, shardID)
		if _, ok := out[status.WorkID]; !ok {
			out[status.WorkID] = map[string]rebalance.RuntimeStatus{}
		}
		out[status.WorkID][status.OwnerNodeID] = status
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateway runtime statuses by work: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ListConsumerRuntimeStatusByWork(ctx context.Context) (map[string]map[string]rebalance.RuntimeStatus, error) {
	rows, err := r.db.Query(ctx, `
		SELECT consumer_id, shard_id, data_plane_id, assignment_state, revoking_done, generation
		FROM smtp.consumer_runtime_statuses
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumer runtime statuses by work: %w", err)
	}
	defer rows.Close()

	out := map[string]map[string]rebalance.RuntimeStatus{}
	for rows.Next() {
		var consumerID string
		var shardID int
		var status rebalance.RuntimeStatus
		if err := rows.Scan(&consumerID, &shardID, &status.OwnerNodeID, &status.AssignmentState, &status.RevokingDone, &status.Generation); err != nil {
			return nil, fmt.Errorf("smtp repo: scan consumer runtime status by work: %w", err)
		}
		status.WorkID = consumerWorkID(consumerID, shardID)
		if _, ok := out[status.WorkID]; !ok {
			out[status.WorkID] = map[string]rebalance.RuntimeStatus{}
		}
		out[status.WorkID][status.OwnerNodeID] = status
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate consumer runtime statuses by work: %w", err)
	}
	return out, nil
}

func (r *RuntimeRepository) ApplyGatewayAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin apply gateway assignments tx: %w", err)
	}
	defer tx.Rollback(ctx)

	workIDs := sortedWorkIDs(rowsByWork)
	for _, workID := range workIDs {
		gatewayID, shardID, ok := parseGatewayWorkID(workID)
		if !ok {
			continue
		}
		if _, err := tx.Exec(ctx, `DELETE FROM smtp.gateway_shard_assignments WHERE gateway_id = $1 AND shard_id = $2`, gatewayID, shardID); err != nil {
			return fmt.Errorf("smtp repo: delete gateway assignment rows: %w", err)
		}

		for _, row := range rowsByWork[workID] {
			if row.OwnerNodeID == "" {
				continue
			}
			if row.AssignmentState == "" {
				row.AssignmentState = leaseassign.StateActive
			}
			if row.DesiredState == "" {
				row.DesiredState = leaseassign.StateActive
			}
			if row.LeaseExpiresAt.IsZero() {
				row.LeaseExpiresAt = time.Now().UTC().Add(30 * time.Second)
			}
			if row.Generation == 0 {
				row.Generation = time.Now().UTC().UnixNano()
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO smtp.gateway_shard_assignments (
					gateway_id, shard_id, data_plane_id, generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			`, gatewayID, shardID, row.OwnerNodeID, row.Generation, row.AssignmentState, row.DesiredState, row.LeaseExpiresAt); err != nil {
				return fmt.Errorf("smtp repo: insert gateway assignment row: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("smtp repo: commit apply gateway assignments tx: %w", err)
	}
	return nil
}

func (r *RuntimeRepository) ApplyConsumerAssignments(ctx context.Context, rowsByWork map[string][]leaseassign.Assignment) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin apply consumer assignments tx: %w", err)
	}
	defer tx.Rollback(ctx)

	workIDs := sortedWorkIDs(rowsByWork)
	for _, workID := range workIDs {
		consumerID, shardID, ok := parseConsumerWorkID(workID)
		if !ok {
			continue
		}
		if _, err := tx.Exec(ctx, `DELETE FROM smtp.consumer_assignments WHERE consumer_id = $1 AND shard_id = $2`, consumerID, shardID); err != nil {
			return fmt.Errorf("smtp repo: delete consumer assignment rows: %w", err)
		}

		for _, row := range rowsByWork[workID] {
			if row.OwnerNodeID == "" {
				continue
			}
			if row.AssignmentState == "" {
				row.AssignmentState = leaseassign.StateActive
			}
			if row.DesiredState == "" {
				row.DesiredState = leaseassign.StateActive
			}
			if row.LeaseExpiresAt.IsZero() {
				row.LeaseExpiresAt = time.Now().UTC().Add(30 * time.Second)
			}
			if row.Generation == 0 {
				row.Generation = time.Now().UTC().UnixNano()
			}

			var targetGatewayID any
			var targetGatewayShardID any
			var targetGatewayDataPlaneID any
			var targetGatewayGRPCEndpoint string
			if row.Metadata != nil {
				if value := strings.TrimSpace(row.Metadata["target_gateway_id"]); value != "" {
					targetGatewayID = value
				}
				if value := strings.TrimSpace(row.Metadata["target_gateway_shard_id"]); value != "" {
					if parsed, err := strconv.Atoi(value); err == nil {
						targetGatewayShardID = parsed
					}
				}
				if value := strings.TrimSpace(row.Metadata["target_gateway_data_plane_id"]); value != "" {
					targetGatewayDataPlaneID = value
				}
				targetGatewayGRPCEndpoint = strings.TrimSpace(row.Metadata["target_gateway_grpc_endpoint"])
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO smtp.consumer_assignments (
					consumer_id, shard_id, data_plane_id, target_gateway_id, target_gateway_shard_id,
					target_gateway_data_plane_id, target_gateway_grpc_endpoint,
					generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7,
					$8, $9, $10, $11, NOW(), NOW()
				)
			`, consumerID, shardID, row.OwnerNodeID, targetGatewayID, targetGatewayShardID, targetGatewayDataPlaneID, targetGatewayGRPCEndpoint, row.Generation, row.AssignmentState, row.DesiredState, row.LeaseExpiresAt); err != nil {
				return fmt.Errorf("smtp repo: insert consumer assignment row: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("smtp repo: commit apply consumer assignments tx: %w", err)
	}
	return nil
}

func chooseGatewayTargetByShard(targets []*gatewayTarget, shardID int) *gatewayTarget {
	if len(targets) == 0 {
		return nil
	}
	ordered := make([]*gatewayTarget, 0, len(targets))
	for _, item := range targets {
		if item == nil {
			continue
		}
		ordered = append(ordered, item)
	}
	if len(ordered) == 0 {
		return nil
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].DataPlaneID == ordered[j].DataPlaneID {
			if ordered[i].GatewayID == ordered[j].GatewayID {
				return ordered[i].ShardID < ordered[j].ShardID
			}
			return ordered[i].GatewayID < ordered[j].GatewayID
		}
		return ordered[i].DataPlaneID < ordered[j].DataPlaneID
	})
	index := shardID % len(ordered)
	if index < 0 {
		index = 0
	}
	return ordered[index]
}

func gatewayWorkID(gatewayID string, shardID int) string {
	return "gateway:" + gatewayID + ":" + strconv.Itoa(shardID)
}

func consumerWorkID(consumerID string, shardID int) string {
	return "consumer:" + consumerID + ":" + strconv.Itoa(shardID)
}

func parseGatewayWorkID(workID string) (string, int, bool) {
	parts := strings.Split(workID, ":")
	if len(parts) != 3 || parts[0] != "gateway" {
		return "", 0, false
	}
	shardID, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, false
	}
	return parts[1], shardID, true
}

func parseConsumerWorkID(workID string) (string, int, bool) {
	parts := strings.Split(workID, ":")
	if len(parts) != 3 || parts[0] != "consumer" {
		return "", 0, false
	}
	shardID, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, false
	}
	return parts[1], shardID, true
}

func sortedWorkIDs(rowsByWork map[string][]leaseassign.Assignment) []string {
	ids := make([]string, 0, len(rowsByWork))
	for workID := range rowsByWork {
		ids = append(ids, workID)
	}
	sort.Strings(ids)
	return ids
}
