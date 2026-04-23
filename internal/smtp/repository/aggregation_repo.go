package repository

import (
	"context"
	"fmt"

	"controlplane/internal/smtp/domain/entity"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AggregationRepository struct {
	db *pgxpool.Pool
}

func NewAggregationRepository(db *pgxpool.Pool) *AggregationRepository {
	return &AggregationRepository{db: db}
}

func (r *AggregationRepository) GetWorkspaceAggregation(ctx context.Context, workspaceID string) (*entity.SMTPOverview, error) {
	overview := &entity.SMTPOverview{}

	// 1. Core Metrics (scalar subqueries to avoid join multiplication)
	if err := r.db.QueryRow(ctx, `
		WITH metrics AS (
			SELECT
				(SELECT COUNT(1) FROM smtp.delivery_attempts WHERE workspace_id = $1 AND created_at >= date_trunc('day', NOW()) AND lower(status) IN ('delivered', 'success', 'sent')) AS delivered_today,
				(SELECT COALESCE(SUM(crs.broker_lag), 0)
				 FROM smtp.consumer_runtime_statuses crs
				 JOIN smtp.consumers c ON c.id = crs.consumer_id
				 WHERE c.workspace_id = $1) AS queued_now,
				(SELECT COUNT(1) FROM smtp.gateways WHERE workspace_id = $1 AND status = 'active') AS active_gateways,
				(SELECT COUNT(1) FROM smtp.gateways WHERE workspace_id = $1) AS total_gateways,
				(SELECT COUNT(1) FROM smtp.templates WHERE workspace_id = $1 AND status = 'live') AS live_templates,
				(SELECT COUNT(1) FROM smtp.templates WHERE workspace_id = $1) AS total_templates
		)
		SELECT delivered_today, queued_now, active_gateways, total_gateways, live_templates, total_templates
		FROM metrics
	`, workspaceID).Scan(
		&overview.Metrics.DeliveredToday,
		&overview.Metrics.QueuedNow,
		&overview.Metrics.ActiveGateways,
		&overview.Metrics.TotalGateways,
		&overview.Metrics.LiveTemplates,
		&overview.Metrics.TotalTemplates,
	); err != nil {
		return nil, fmt.Errorf("smtp repo: get aggregation metrics: %w", err)
	}

	// 2. Throughput (Biểu đồ 7 ngày - QUAN TRỌNG: Chỉ quét bảng da 1 lần)
	throughputRows, err := r.db.Query(ctx, `
		WITH days AS (
			SELECT generate_series(date_trunc('day', NOW()) - interval '6 day', date_trunc('day', NOW()), interval '1 day') AS day
		),
		stats AS (
			SELECT 
				date_trunc('day', created_at) as day,
				COUNT(1) FILTER (WHERE lower(status) IN ('delivered', 'success', 'sent')) as delivered,
				COUNT(1) FILTER (WHERE lower(status) IN ('queued', 'pending', 'processing')) as queued,
				COUNT(1) FILTER (WHERE retry_count > 0) as retries
			FROM smtp.delivery_attempts
			WHERE workspace_id = $1 
			  AND created_at >= date_trunc('day', NOW()) - interval '6 day'
			GROUP BY 1
		)
		SELECT 
			to_char(d.day, 'Mon DD') AS label,
			COALESCE(s.delivered, 0),
			COALESCE(s.queued, 0),
			COALESCE(s.retries, 0)
		FROM days d
		LEFT JOIN stats s ON d.day = s.day
		ORDER BY d.day ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list aggregation throughput: %w", err)
	}
	defer throughputRows.Close()

	for throughputRows.Next() {
		var item entity.OverviewThroughputPoint
		if err := throughputRows.Scan(&item.Label, &item.Delivered, &item.Queued, &item.Retries); err != nil {
			return nil, fmt.Errorf("smtp repo: scan aggregation throughput: %w", err)
		}
		overview.DeliveryThroughput = append(overview.DeliveryThroughput, &item)
	}

	// 3. Health Distribution
	if err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(1) FILTER (WHERE status = 'active') AS healthy,
			COUNT(1) FILTER (WHERE status = 'draining') AS warning,
			COUNT(1) FILTER (WHERE status = 'disabled') AS stopped
		FROM smtp.gateways
		WHERE workspace_id = $1
	`, workspaceID).Scan(
		&overview.HealthDistribution.Healthy,
		&overview.HealthDistribution.Warning,
		&overview.HealthDistribution.Stopped,
	); err != nil {
		return nil, fmt.Errorf("smtp repo: get aggregation health distribution: %w", err)
	}

	// 4. Queue Mix (Gộp logic retries vào GROUP BY)
	queueMixRows, err := r.db.Query(ctx, `
		SELECT
			c.transport_type::text AS category,
			COALESCE(SUM(crs.broker_lag), 0) AS pending,
			COALESCE(SUM(crs.inflight_count), 0) AS processing,
			(SELECT COUNT(1) FROM smtp.delivery_attempts da WHERE da.workspace_id = $1 AND da.retry_count > 0 AND EXISTS (SELECT 1 FROM smtp.consumers c2 WHERE c2.id = da.consumer_id AND c2.transport_type = c.transport_type)) as retries
		FROM smtp.consumers c
		LEFT JOIN smtp.consumer_runtime_statuses crs ON crs.consumer_id = c.id
		WHERE c.workspace_id = $1
		GROUP BY c.transport_type
		ORDER BY c.transport_type ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list aggregation queue mix: %w", err)
	}
	defer queueMixRows.Close()

	for queueMixRows.Next() {
		var item entity.OverviewQueueMixItem
		if err := queueMixRows.Scan(&item.Category, &item.Pending, &item.Processing, &item.Retries); err != nil {
			return nil, fmt.Errorf("smtp repo: scan aggregation queue mix: %w", err)
		}
		overview.QueueMix = append(overview.QueueMix, &item)
	}

	// 5. Gateways (Tối ưu Join bằng cách giới hạn Gateway trước khi Join metadata)
	gatewayRows, err := r.db.Query(ctx, `
		WITH target_gateways AS (
			SELECT * FROM smtp.gateways 
			WHERE workspace_id = $1 
			ORDER BY updated_at DESC, id DESC 
			LIMIT 5
		)
		SELECT
			g.id, g.name, g.traffic_class, g.status::text, g.routing_mode, g.priority, g.desired_shard_count,
			COALESCE(tc.template_count, 0),
			COALESCE(ec.endpoint_count, 0),
			COALESCE(sc.ready_shards, 0),
			COALESCE(sc.pending_shards, 0),
			COALESCE(sc.draining_shards, 0),
			COALESCE(fg.name, ''),
			g.updated_at
		FROM target_gateways g
		LEFT JOIN smtp.gateways fg ON fg.id = g.fallback_gateway_id
		LEFT JOIN LATERAL (SELECT COUNT(1) as template_count FROM smtp.gateway_templates WHERE gateway_id = g.id AND enabled = TRUE) tc ON TRUE
		LEFT JOIN LATERAL (SELECT COUNT(1) as endpoint_count FROM smtp.gateway_endpoints WHERE gateway_id = g.id AND enabled = TRUE) ec ON TRUE
		LEFT JOIN LATERAL (
			SELECT
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'active') AS ready_shards,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'pending') AS pending_shards,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'revoking') AS draining_shards
			FROM smtp.gateway_shard_assignments
			WHERE gateway_id = g.id
		) sc ON TRUE
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list aggregation gateways: %w", err)
	}
	defer gatewayRows.Close()

	for gatewayRows.Next() {
		var item entity.GatewayListItem
		if err := gatewayRows.Scan(&item.ID, &item.Name, &item.TrafficClass, &item.Status, &item.RoutingMode, &item.Priority, &item.DesiredShardCount, &item.TemplateCount, &item.EndpointCount, &item.ReadyShards, &item.PendingShards, &item.DrainingShards, &item.FallbackGatewayName, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan aggregation gateway: %w", err)
		}
		overview.Gateways = append(overview.Gateways, &item)
	}

	// 6. Timeline (Dùng workspace_id trực tiếp để index scan)
	timelineRows, err := r.db.Query(ctx, `
		SELECT id, entity_type, entity_name, action, actor_name, note, created_at
		FROM smtp.activity_logs
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT 10
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list aggregation timeline: %w", err)
	}
	defer timelineRows.Close()

	for timelineRows.Next() {
		var item entity.OverviewTimelineItem
		if err := timelineRows.Scan(&item.ID, &item.EntityType, &item.EntityName, &item.Action, &item.ActorName, &item.Note, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("smtp repo: scan aggregation timeline: %w", err)
		}
		overview.Timeline = append(overview.Timeline, &item)
	}

	return overview, nil
}
