package repository

import (
	"context"
	"errors"
	"fmt"

	"controlplane/internal/smtp/domain/entity"
	smtp_errorx "controlplane/internal/smtp/errorx"
	smtp_model "controlplane/internal/smtp/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RuntimeRepository struct {
	db *pgxpool.Pool
}

func NewRuntimeRepository(db *pgxpool.Pool) *RuntimeRepository {
	return &RuntimeRepository{db: db}
}

func (r *RuntimeRepository) ListActivityLogs(ctx context.Context, workspaceID string) ([]*entity.ActivityLog, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, entity_type, entity_id, entity_name, action, actor_name, note, workspace_id, created_at
		FROM smtp.activity_logs
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT 200
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list activity logs: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.ActivityLog, 0)
	for rows.Next() {
		var row smtp_model.ActivityLog
		if err := rows.Scan(&row.ID, &row.EntityType, &row.EntityID, &row.EntityName, &row.Action, &row.ActorName, &row.Note, &row.WorkspaceID, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan activity log: %w", err)
		}
		items = append(items, smtp_model.ActivityLogModelToEntity(&row))
	}
	return items, rows.Err()
}

func (r *RuntimeRepository) ListDeliveryAttempts(ctx context.Context, workspaceID string) ([]*entity.DeliveryAttempt, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, consumer_id, template_id, gateway_id, endpoint_id, message_id, transport_message_id,
		       subject, status, error_message, error_class, retry_count, trace_id, payload, workspace_id, created_at
		FROM smtp.delivery_attempts
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT 200
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list delivery attempts: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.DeliveryAttempt, 0)
	for rows.Next() {
		var row smtp_model.DeliveryAttempt
		if err := rows.Scan(&row.ID, &row.ConsumerID, &row.TemplateID, &row.GatewayID, &row.EndpointID, &row.MessageID, &row.TransportMessageID, &row.Subject, &row.Status, &row.ErrorMessage, &row.ErrorClass, &row.RetryCount, &row.TraceID, &row.Payload, &row.WorkspaceID, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan delivery attempt: %w", err)
		}
		items = append(items, smtp_model.DeliveryAttemptModelToEntity(&row))
	}
	return items, rows.Err()
}

func (r *RuntimeRepository) ListRuntimeHeartbeats(ctx context.Context) ([]*entity.RuntimeHeartbeat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT data_plane_id, sent_at, local_version, gateway_count, consumer_count, member_state, capacity, grpc_addr, updated_at
		FROM smtp.runtime_heartbeats
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list runtime heartbeats: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.RuntimeHeartbeat, 0)
	for rows.Next() {
		var row smtp_model.RuntimeHeartbeat
		if err := rows.Scan(&row.DataPlaneID, &row.SentAt, &row.LocalVersion, &row.GatewayCount, &row.ConsumerCount, &row.MemberState, &row.Capacity, &row.GRPCAddr, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan runtime heartbeat: %w", err)
		}
		items = append(items, smtp_model.RuntimeHeartbeatModelToEntity(&row))
	}
	return items, rows.Err()
}

func (r *RuntimeRepository) ListGatewayAssignments(ctx context.Context) ([]*entity.GatewayShardAssignment, error) {
	return r.listGatewayAssignments(ctx, `
		SELECT gsa.gateway_id, gsa.shard_id, gsa.data_plane_id, COALESCE(dp.grpc_endpoint, ''),
		       gsa.generation, gsa.assignment_state, gsa.desired_state, gsa.lease_expires_at, gsa.assigned_at, gsa.updated_at
		FROM smtp.gateway_shard_assignments gsa
		LEFT JOIN core.data_planes dp ON dp.id = gsa.data_plane_id
		ORDER BY gateway_id, shard_id, updated_at DESC
	`)
}

func (r *RuntimeRepository) ListConsumerAssignments(ctx context.Context) ([]*entity.ConsumerShardAssignment, error) {
	return r.listConsumerAssignments(ctx, `
		SELECT consumer_id, shard_id, data_plane_id, target_gateway_id, target_gateway_shard_id,
		       target_gateway_data_plane_id, target_gateway_grpc_endpoint,
		       generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at
		FROM smtp.consumer_assignments
		ORDER BY consumer_id, shard_id, updated_at DESC
	`)
}

func (r *RuntimeRepository) GetRuntimeDataPlane(ctx context.Context, dataPlaneID string) (*entity.RuntimeDataPlane, error) {
	var row smtp_model.RuntimeDataPlane
	err := r.db.QueryRow(ctx, `
		SELECT dp.id, dp.zone_id, dp.grpc_endpoint, dp.status, dp.last_seen_at, COALESCE(h.capacity, 1)
		FROM core.data_planes dp
		LEFT JOIN smtp.runtime_heartbeats h ON h.data_plane_id = dp.id
		WHERE dp.id = $1
	`, dataPlaneID).Scan(&row.ID, &row.ZoneID, &row.GRPCEndpoint, &row.Status, &row.LastSeenAt, &row.Capacity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrDataPlaneNotFound
		}
		return nil, fmt.Errorf("smtp repo: get runtime dataplane: %w", err)
	}
	return smtp_model.RuntimeDataPlaneModelToEntity(&row), nil
}

func (r *RuntimeRepository) ListHealthyRuntimeDataPlanesByZone(ctx context.Context, zoneID string) ([]*entity.RuntimeDataPlane, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dp.id, dp.zone_id, dp.grpc_endpoint, dp.status, dp.last_seen_at, COALESCE(h.capacity, 1)
		FROM core.data_planes dp
		LEFT JOIN smtp.runtime_heartbeats h ON h.data_plane_id = dp.id
		WHERE dp.zone_id = $1
		  AND dp.status = 'healthy'
		ORDER BY COALESCE(h.capacity, 1) DESC, dp.last_seen_at DESC NULLS LAST, dp.created_at ASC
	`, zoneID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list healthy runtime dataplanes by zone: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.RuntimeDataPlane, 0)
	for rows.Next() {
		var row smtp_model.RuntimeDataPlane
		if err := rows.Scan(&row.ID, &row.ZoneID, &row.GRPCEndpoint, &row.Status, &row.LastSeenAt, &row.Capacity); err != nil {
			return nil, fmt.Errorf("smtp repo: scan healthy runtime dataplane by zone: %w", err)
		}
		items = append(items, smtp_model.RuntimeDataPlaneModelToEntity(&row))
	}
	return items, rows.Err()
}

func (r *RuntimeRepository) UpsertRuntimeHeartbeat(ctx context.Context, heartbeat *entity.RuntimeHeartbeat) error {
	if heartbeat == nil {
		return smtp_errorx.ErrRuntimeInvalid
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO smtp.runtime_heartbeats (
			data_plane_id, sent_at, local_version, gateway_count, consumer_count, member_state, capacity, grpc_addr, updated_at
		) VALUES (
			$1, NOW(), $2, $3, $4, $5, $6, $7, NOW()
		)
		ON CONFLICT (data_plane_id) DO UPDATE SET
			sent_at = NOW(),
			local_version = EXCLUDED.local_version,
			gateway_count = EXCLUDED.gateway_count,
			consumer_count = EXCLUDED.consumer_count,
			member_state = EXCLUDED.member_state,
			capacity = EXCLUDED.capacity,
			grpc_addr = EXCLUDED.grpc_addr,
			updated_at = NOW()
	`, heartbeat.DataPlaneID, heartbeat.LocalVersion, heartbeat.GatewayCount, heartbeat.ConsumerCount, heartbeat.MemberState, maxInt(heartbeat.Capacity, 1), heartbeat.GRPCAddr)
	if err != nil {
		return fmt.Errorf("smtp repo: upsert runtime heartbeat: %w", err)
	}
	return nil
}

func (r *RuntimeRepository) ReplaceGatewayStatuses(ctx context.Context, dataPlaneID string, statuses []*entity.GatewayShardStatus) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin gateway statuses tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM smtp.gateway_runtime_statuses WHERE data_plane_id = $1`, dataPlaneID); err != nil {
		return fmt.Errorf("smtp repo: delete gateway runtime statuses: %w", err)
	}

	for _, status := range statuses {
		if status == nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO smtp.gateway_runtime_statuses (
				gateway_id, shard_id, data_plane_id, status, inflight_count, desired_workers, active_workers,
				relay_queue_depth, pool_open_conns, pool_busy_conns, send_rate_per_second, backpressure_state,
				last_error, version, generation, assignment_state, revoking_done, last_report_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 0, $14, $15, $16, NOW()
			)
		`, status.GatewayID, status.ShardID, dataPlaneID, status.Status, status.InflightCount, status.DesiredWorkers, status.ActiveWorkers, status.RelayQueueDepth, status.PoolOpenConns, status.PoolBusyConns, status.SendRate, status.Backpressure, status.LastError, status.Generation, status.AssignmentState, status.RevokingDone); err != nil {
			return fmt.Errorf("smtp repo: insert gateway runtime status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *RuntimeRepository) ReplaceConsumerStatuses(ctx context.Context, dataPlaneID string, statuses []*entity.ConsumerShardStatus) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin consumer statuses tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM smtp.consumer_runtime_statuses WHERE data_plane_id = $1`, dataPlaneID); err != nil {
		return fmt.Errorf("smtp repo: delete consumer runtime statuses: %w", err)
	}

	for _, status := range statuses {
		if status == nil {
			continue
		}
		var gatewayID any
		if status.GatewayID != "" {
			gatewayID = status.GatewayID
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO smtp.consumer_runtime_statuses (
				consumer_id, shard_id, data_plane_id, gateway_id, status, inflight_count, broker_lag,
				oldest_unacked_age_ms, desired_workers, active_workers, relay_queue_depth, last_error,
				version, generation, assignment_state, revoking_done, last_report_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 0, $13, $14, $15, NOW()
			)
		`, status.ConsumerID, status.ShardID, dataPlaneID, gatewayID, status.Status, status.InflightCount, status.BrokerLag, status.OldestUnackedMS, status.DesiredWorkers, status.ActiveWorkers, status.RelayQueueDepth, status.LastError, status.Generation, status.AssignmentState, status.RevokingDone); err != nil {
			return fmt.Errorf("smtp repo: insert consumer runtime status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *RuntimeRepository) ReconcileAssignments(ctx context.Context) error {
	return r.ensureDesiredShards(ctx)
}

func (r *RuntimeRepository) EnsureDesiredShards(ctx context.Context) error {
	return r.ensureDesiredShards(ctx)
}

func (r *RuntimeRepository) ListGatewayAssignmentsByDataPlane(ctx context.Context, dataPlaneID string) ([]*entity.GatewayShardAssignment, error) {
	return r.listGatewayAssignments(ctx, `
		SELECT gsa.gateway_id, gsa.shard_id, gsa.data_plane_id, COALESCE(dp.grpc_endpoint, ''),
		       gsa.generation, gsa.assignment_state, gsa.desired_state, gsa.lease_expires_at, gsa.assigned_at, gsa.updated_at
		FROM smtp.gateway_shard_assignments gsa
		LEFT JOIN core.data_planes dp ON dp.id = gsa.data_plane_id
		WHERE gsa.data_plane_id = $1
		ORDER BY gateway_id, shard_id
	`, dataPlaneID)
}

func (r *RuntimeRepository) ListConsumerAssignmentsByDataPlane(ctx context.Context, dataPlaneID string) ([]*entity.ConsumerShardAssignment, error) {
	return r.listConsumerAssignments(ctx, `
		SELECT consumer_id, shard_id, data_plane_id, target_gateway_id, target_gateway_shard_id,
		       target_gateway_data_plane_id, target_gateway_grpc_endpoint,
		       generation, assignment_state, desired_state, lease_expires_at, assigned_at, updated_at
		FROM smtp.consumer_assignments
		WHERE data_plane_id = $1
		ORDER BY consumer_id, shard_id
	`, dataPlaneID)
}

func (r *RuntimeRepository) ListWorkspaceIDsByDataPlane(ctx context.Context, dataPlaneID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH assigned_workspaces AS (
			SELECT DISTINCT c.workspace_id
			FROM smtp.consumer_assignments ca
			JOIN smtp.consumers c ON c.id = ca.consumer_id
			WHERE ca.data_plane_id = $1
			UNION
			SELECT DISTINCT g.workspace_id
			FROM smtp.gateway_shard_assignments gsa
			JOIN smtp.gateways g ON g.id = gsa.gateway_id
			WHERE gsa.data_plane_id = $1
		)
		SELECT workspace_id
		FROM assigned_workspaces
		WHERE workspace_id IS NOT NULL AND workspace_id <> ''
		ORDER BY workspace_id
	`, dataPlaneID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list runtime workspaces by dataplane: %w", err)
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var workspaceID string
		if err := rows.Scan(&workspaceID); err != nil {
			return nil, fmt.Errorf("smtp repo: scan runtime workspace: %w", err)
		}
		items = append(items, workspaceID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate runtime workspaces: %w", err)
	}
	return items, nil
}

func (r *RuntimeRepository) GetRuntimeVersionByDataPlane(ctx context.Context, dataPlaneID string) (int64, error) {
	var version int64
	err := r.db.QueryRow(ctx, `
		WITH assigned_workspaces AS (
			SELECT DISTINCT c.workspace_id
			FROM smtp.consumer_assignments ca
			JOIN smtp.consumers c ON c.id = ca.consumer_id
			WHERE ca.data_plane_id = $1
			UNION
			SELECT DISTINCT g.workspace_id
			FROM smtp.gateway_shard_assignments gsa
			JOIN smtp.gateways g ON g.id = gsa.gateway_id
			WHERE gsa.data_plane_id = $1
		),
		resource_versions AS (
			SELECT COALESCE(MAX(c.runtime_version), 0) AS version
			FROM smtp.consumer_assignments ca
			JOIN smtp.consumers c ON c.id = ca.consumer_id
			WHERE ca.data_plane_id = $1
			UNION ALL
			SELECT COALESCE(MAX(g.runtime_version), 0) AS version
			FROM smtp.gateway_shard_assignments gsa
			JOIN smtp.gateways g ON g.id = gsa.gateway_id
			WHERE gsa.data_plane_id = $1
			UNION ALL
			SELECT COALESCE(MAX(t.runtime_version), 0) AS version
			FROM smtp.templates t
			WHERE t.workspace_id IN (SELECT workspace_id FROM assigned_workspaces)
			UNION ALL
			SELECT COALESCE(MAX(e.runtime_version), 0) AS version
			FROM smtp.endpoints e
			WHERE e.workspace_id IN (SELECT workspace_id FROM assigned_workspaces)
			UNION ALL
			SELECT COALESCE(MAX(ca.generation), 0) AS version
			FROM smtp.consumer_assignments ca
			WHERE ca.data_plane_id = $1
			UNION ALL
			SELECT COALESCE(MAX(gsa.generation), 0) AS version
			FROM smtp.gateway_shard_assignments gsa
			WHERE gsa.data_plane_id = $1
		)
		SELECT COALESCE(MAX(version), 0) FROM resource_versions
	`, dataPlaneID).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("smtp repo: get runtime version by dataplane: %w", err)
	}
	return version, nil
}

func (r *RuntimeRepository) ensureDesiredShards(ctx context.Context) error {
	rows, err := r.db.Query(ctx, `SELECT id, desired_shard_count FROM smtp.gateways WHERE status = 'active'`)
	if err != nil {
		return fmt.Errorf("smtp repo: list active gateways for shards: %w", err)
	}
	type shardTarget struct {
		id      string
		desired int
	}
	var gatewayTargets []shardTarget
	for rows.Next() {
		var t shardTarget
		if err := rows.Scan(&t.id, &t.desired); err != nil {
			rows.Close()
			return fmt.Errorf("smtp repo: scan active gateway shard target: %w", err)
		}
		gatewayTargets = append(gatewayTargets, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("smtp repo: iterate active gateway shard targets: %w", err)
	}

	for _, t := range gatewayTargets {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("smtp repo: begin gateway shard tx: %w", err)
		}
		if err := syncGatewayShards(ctx, tx, t.id, maxInt(t.desired, 1)); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("smtp repo: commit gateway shard tx: %w", err)
		}
	}

	rows, err = r.db.Query(ctx, `SELECT id, desired_shard_count FROM smtp.consumers WHERE status = 'active'`)
	if err != nil {
		return fmt.Errorf("smtp repo: list active consumers for shards: %w", err)
	}
	var consumerTargets []shardTarget
	for rows.Next() {
		var t shardTarget
		if err := rows.Scan(&t.id, &t.desired); err != nil {
			rows.Close()
			return fmt.Errorf("smtp repo: scan active consumer shard target: %w", err)
		}
		consumerTargets = append(consumerTargets, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("smtp repo: iterate active consumer shard targets: %w", err)
	}

	for _, t := range consumerTargets {
		tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("smtp repo: begin consumer shard tx: %w", err)
		}
		if err := syncConsumerShards(ctx, tx, t.id, maxInt(t.desired, 1)); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("smtp repo: commit consumer shard tx: %w", err)
		}
	}

	return nil
}

type gatewayTarget struct {
	ConsumerID   string
	GatewayID    string
	ShardID      int
	DataPlaneID  string
	GRPCEndpoint string
	ZoneID       string
}

func (r *RuntimeRepository) compatibleGatewayTargetsByConsumer(ctx context.Context) (map[string][]*gatewayTarget, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT t.consumer_id, gsa.gateway_id, gsa.shard_id, gsa.data_plane_id, COALESCE(dp.grpc_endpoint, ''), g.zone_id
		FROM smtp.templates t
		JOIN smtp.consumers c ON c.id = t.consumer_id
		JOIN smtp.gateway_templates gt ON gt.template_id = t.id AND gt.enabled = TRUE
		JOIN smtp.gateways g ON g.id = gt.gateway_id
		JOIN smtp.gateway_shard_assignments gsa ON gsa.gateway_id = g.id
		JOIN core.data_planes dp ON dp.id = gsa.data_plane_id
		WHERE t.consumer_id IS NOT NULL
		  AND t.status = 'live'
		  AND c.status = 'active'
		  AND g.status = 'active'
		  AND gsa.assignment_state = 'active'
		  AND gsa.desired_state = 'active'
		  AND dp.status = 'healthy'
		  AND c.zone_id IS NOT NULL
		  AND g.zone_id = c.zone_id
	`)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list compatible gateway targets: %w", err)
	}
	defer rows.Close()

	items := make(map[string][]*gatewayTarget)
	for rows.Next() {
		var target gatewayTarget
		if err := rows.Scan(&target.ConsumerID, &target.GatewayID, &target.ShardID, &target.DataPlaneID, &target.GRPCEndpoint, &target.ZoneID); err != nil {
			return nil, fmt.Errorf("smtp repo: scan compatible gateway target: %w", err)
		}
		items[target.ConsumerID] = append(items[target.ConsumerID], &target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate compatible gateway targets: %w", err)
	}
	return items, nil
}

func (r *RuntimeRepository) listGatewayAssignments(ctx context.Context, query string, args ...any) ([]*entity.GatewayShardAssignment, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway assignments: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.GatewayShardAssignment, 0)
	for rows.Next() {
		var row smtp_model.GatewayShardAssignment
		if err := rows.Scan(&row.GatewayID, &row.ShardID, &row.DataPlaneID, &row.GRPCEndpoint, &row.Generation, &row.AssignmentState, &row.DesiredState, &row.LeaseExpiresAt, &row.AssignedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway assignment: %w", err)
		}
		items = append(items, smtp_model.GatewayShardAssignmentModelToEntity(&row))
	}
	return items, rows.Err()
}

func (r *RuntimeRepository) listConsumerAssignments(ctx context.Context, query string, args ...any) ([]*entity.ConsumerShardAssignment, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumer assignments: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.ConsumerShardAssignment, 0)
	for rows.Next() {
		var row smtp_model.ConsumerShardAssignment
		if err := rows.Scan(&row.ConsumerID, &row.ShardID, &row.DataPlaneID, &row.TargetGatewayID, &row.TargetShardID, &row.TargetPlaneID, &row.TargetGRPCAddr, &row.Generation, &row.AssignmentState, &row.DesiredState, &row.LeaseExpiresAt, &row.AssignedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan consumer assignment: %w", err)
		}
		items = append(items, smtp_model.ConsumerShardAssignmentModelToEntity(&row))
	}
	return items, rows.Err()
}
