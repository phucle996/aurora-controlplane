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

type GatewayRepository struct {
	db *pgxpool.Pool
}

func NewGatewayRepository(db *pgxpool.Pool) *GatewayRepository {
	return &GatewayRepository{db: db}
}

func (r *GatewayRepository) ListGatewayItemsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.GatewayListItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			g.id,
			g.name,
			g.traffic_class,
			g.status::text,
			g.routing_mode,
			g.priority,
			g.desired_shard_count,
			COALESCE(tc.template_count, 0) AS template_count,
			COALESCE(ec.endpoint_count, 0) AS endpoint_count,
			COALESCE(sc.ready_shards, 0) AS ready_shards,
			COALESCE(sc.pending_shards, 0) AS pending_shards,
			COALESCE(sc.draining_shards, 0) AS draining_shards,
			COALESCE(fg.name, '') AS fallback_gateway_name,
			g.updated_at
		FROM smtp.gateways g
		LEFT JOIN smtp.gateways fg ON fg.id = g.fallback_gateway_id
		LEFT JOIN (
			SELECT gateway_id, COUNT(1) FILTER (WHERE enabled = TRUE) AS template_count
			FROM smtp.gateway_templates
			GROUP BY gateway_id
		) tc ON tc.gateway_id = g.id
		LEFT JOIN (
			SELECT gateway_id, COUNT(1) FILTER (WHERE enabled = TRUE) AS endpoint_count
			FROM smtp.gateway_endpoints
			GROUP BY gateway_id
		) ec ON ec.gateway_id = g.id
		LEFT JOIN (
			SELECT
				gateway_id,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'active') AS ready_shards,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'pending') AS pending_shards,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'revoking') AS draining_shards
			FROM smtp.gateway_shard_assignments
			GROUP BY gateway_id
		) sc ON sc.gateway_id = g.id
		WHERE g.workspace_id = $1
		ORDER BY g.created_at DESC, g.id DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway items: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.GatewayListItem, 0)
	for rows.Next() {
		var item entity.GatewayListItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.TrafficClass,
			&item.Status,
			&item.RoutingMode,
			&item.Priority,
			&item.DesiredShardCount,
			&item.TemplateCount,
			&item.EndpointCount,
			&item.ReadyShards,
			&item.PendingShards,
			&item.DrainingShards,
			&item.FallbackGatewayName,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway item: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateway items: %w", err)
	}
	return items, nil
}

func (r *GatewayRepository) GetGatewayDetail(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	var item entity.GatewayDetail
	var (
		fallbackID     string
		fallbackName   string
		fallbackStatus string
		zoneID         string
	)
	err := r.db.QueryRow(ctx, `
		SELECT
			g.id,
			g.name,
			g.traffic_class,
			g.status::text,
			g.routing_mode,
			g.priority,
			g.desired_shard_count,
			g.runtime_version,
			g.created_at,
			g.updated_at,
			COALESCE(g.zone_id, '') AS zone_id,
			COALESCE(fg.id, '') AS fallback_id,
			COALESCE(fg.name, '') AS fallback_name,
			COALESCE(fg.status::text, '') AS fallback_status,
			COALESCE(sc.ready_shards, 0) AS ready_shards,
			COALESCE(sc.pending_shards, 0) AS pending_shards,
			COALESCE(sc.draining_shards, 0) AS draining_shards
		FROM smtp.gateways g
		LEFT JOIN smtp.gateways fg ON fg.id = g.fallback_gateway_id
		LEFT JOIN (
			SELECT
				gateway_id,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'active') AS ready_shards,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'pending') AS pending_shards,
				COUNT(DISTINCT shard_id) FILTER (WHERE assignment_state = 'revoking') AS draining_shards
			FROM smtp.gateway_shard_assignments
			GROUP BY gateway_id
		) sc ON sc.gateway_id = g.id
		WHERE g.workspace_id = $1 AND g.id = $2
	`, workspaceID, gatewayID).Scan(
		&item.ID,
		&item.Name,
		&item.TrafficClass,
		&item.Status,
		&item.RoutingMode,
		&item.Priority,
		&item.DesiredShardCount,
		&item.RuntimeVersion,
		&item.CreatedAt,
		&item.UpdatedAt,
		&zoneID,
		&fallbackID,
		&fallbackName,
		&fallbackStatus,
		&item.ReadyShards,
		&item.PendingShards,
		&item.DrainingShards,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrGatewayNotFound
		}
		return nil, fmt.Errorf("smtp repo: get gateway detail: %w", err)
	}
	if fallbackID != "" {
		item.FallbackGateway = &entity.GatewayFallbackSummary{
			ID:     fallbackID,
			Name:   fallbackName,
			Status: fallbackStatus,
		}
	}

	templateRows, err := r.db.Query(ctx, `
		SELECT
			t.id,
			t.name,
			t.category,
			t.traffic_class,
			t.status::text,
			COALESCE(t.consumer_id, ''),
			COALESCE(c.name, ''),
			(gt.template_id IS NOT NULL) AS selected,
			COALESCE(gt.position, 0) AS position
		FROM smtp.templates t
		LEFT JOIN smtp.consumers c ON c.id = t.consumer_id
		LEFT JOIN smtp.gateway_templates gt ON gt.gateway_id = $2 AND gt.template_id = t.id AND gt.enabled = TRUE
		WHERE t.workspace_id = $1
		  AND (
			t.consumer_id IS NULL
			OR COALESCE(c.zone_id, '') = ''
			OR COALESCE(c.zone_id, '') = $3
		  )
		ORDER BY
			CASE WHEN gt.template_id IS NULL THEN 1 ELSE 0 END,
			COALESCE(gt.position, 2147483647),
			t.name ASC,
			t.id ASC
	`, workspaceID, gatewayID, zoneID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway detail templates: %w", err)
	}
	defer templateRows.Close()

	item.Templates = make([]*entity.GatewayTemplateBinding, 0)
	for templateRows.Next() {
		var row entity.GatewayTemplateBinding
		if err := templateRows.Scan(
			&row.ID,
			&row.Name,
			&row.Category,
			&row.TrafficClass,
			&row.Status,
			&row.ConsumerID,
			&row.ConsumerName,
			&row.Selected,
			&row.Position,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway detail template: %w", err)
		}
		item.Templates = append(item.Templates, &row)
	}
	if err := templateRows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateway detail templates: %w", err)
	}

	endpointRows, err := r.db.Query(ctx, `
		SELECT
			e.id,
			e.name,
			e.host,
			e.port,
			e.username,
			e.status::text,
			(ge.endpoint_id IS NOT NULL) AS selected,
			COALESCE(ge.position, 0) AS position
		FROM smtp.endpoints e
		LEFT JOIN smtp.gateway_endpoints ge ON ge.gateway_id = $2 AND ge.endpoint_id = e.id AND ge.enabled = TRUE
		WHERE e.workspace_id = $1
		ORDER BY
			CASE WHEN ge.endpoint_id IS NULL THEN 1 ELSE 0 END,
			COALESCE(ge.position, 2147483647),
			e.name ASC,
			e.id ASC
	`, workspaceID, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateway detail endpoints: %w", err)
	}
	defer endpointRows.Close()

	item.Endpoints = make([]*entity.GatewayEndpointBinding, 0)
	for endpointRows.Next() {
		var row entity.GatewayEndpointBinding
		if err := endpointRows.Scan(
			&row.ID,
			&row.Name,
			&row.Host,
			&row.Port,
			&row.Username,
			&row.Status,
			&row.Selected,
			&row.Position,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan gateway detail endpoint: %w", err)
		}
		item.Endpoints = append(item.Endpoints, &row)
	}
	if err := endpointRows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateway detail endpoints: %w", err)
	}

	return &item, nil
}

func (r *GatewayRepository) ListGatewaysByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Gateway, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, workspace_id, owner_user_id, zone_id, name, traffic_class, status::text,
			routing_mode, priority, fallback_gateway_id, runtime_version, desired_shard_count, created_at, updated_at
		FROM smtp.gateways
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list gateways: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.Gateway, 0)
	for rows.Next() {
		row, err := scanGateway(rows)
		if err != nil {
			return nil, err
		}
		if err := r.populateGatewayRelations(ctx, row); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate gateways: %w", err)
	}
	return items, nil
}

func (r *GatewayRepository) GetGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.Gateway, error) {
	row, err := r.getGatewayByQuery(ctx, `
		SELECT
			id, workspace_id, owner_user_id, zone_id, name, traffic_class, status::text,
			routing_mode, priority, fallback_gateway_id, runtime_version, desired_shard_count, created_at, updated_at
		FROM smtp.gateways
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, gatewayID)
	if err != nil {
		return nil, err
	}
	return row, r.populateGatewayRelations(ctx, row)
}

func (r *GatewayRepository) GetGatewayByID(ctx context.Context, gatewayID string) (*entity.Gateway, error) {
	row, err := r.getGatewayByQuery(ctx, `
		SELECT
			id, workspace_id, owner_user_id, zone_id, name, traffic_class, status::text,
			routing_mode, priority, fallback_gateway_id, runtime_version, desired_shard_count, created_at, updated_at
		FROM smtp.gateways
		WHERE id = $1
	`, gatewayID)
	if err != nil {
		return nil, err
	}
	return row, r.populateGatewayRelations(ctx, row)
}

func (r *GatewayRepository) CreateGateway(ctx context.Context, gateway *entity.Gateway) error {
	if gateway == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin create gateway tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.GatewayEntityToModel(gateway)
	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.gateways (
			id, workspace_id, owner_user_id, zone_id, name, traffic_class, status,
			routing_mode, priority, fallback_gateway_id, desired_shard_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::smtp.gateway_status,
			$8, $9, $10, $11, NOW(), NOW()
		)
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.ZoneID, row.Name, row.TrafficClass, row.Status,
		row.RoutingMode, row.Priority, row.FallbackGatewayID, maxInt(row.DesiredShardCount, 1),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: create gateway: %w", err)
	}

	if err := syncGatewayTemplates(ctx, tx, row.ID, gateway.TemplateIDs); err != nil {
		return err
	}
	if err := syncGatewayEndpoints(ctx, tx, row.ID, gateway.EndpointIDs); err != nil {
		return err
	}
	if err := syncGatewayShards(ctx, tx, row.ID, maxInt(row.DesiredShardCount, 1)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *GatewayRepository) UpdateGateway(ctx context.Context, gateway *entity.Gateway) error {
	if gateway == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin update gateway tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.GatewayEntityToModel(gateway)
	tag, err := tx.Exec(ctx, `
		UPDATE smtp.gateways
		SET
			workspace_id = $2,
			owner_user_id = $3,
			zone_id = $4,
			name = $5,
			traffic_class = $6,
			status = $7::smtp.gateway_status,
			routing_mode = $8,
			priority = $9,
			fallback_gateway_id = $10,
			desired_shard_count = $11,
			updated_at = NOW()
		WHERE id = $1
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.ZoneID, row.Name, row.TrafficClass, row.Status, row.RoutingMode,
		row.Priority, row.FallbackGatewayID, maxInt(row.DesiredShardCount, 1),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: update gateway: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrGatewayNotFound
	}

	if err := syncGatewayTemplates(ctx, tx, row.ID, gateway.TemplateIDs); err != nil {
		return err
	}
	if err := syncGatewayEndpoints(ctx, tx, row.ID, gateway.EndpointIDs); err != nil {
		return err
	}
	if err := syncGatewayShards(ctx, tx, row.ID, maxInt(row.DesiredShardCount, 1)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *GatewayRepository) DeleteGateway(ctx context.Context, workspaceID, gatewayID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM smtp.gateways WHERE workspace_id = $1 AND id = $2`, workspaceID, gatewayID)
	if err != nil {
		return fmt.Errorf("smtp repo: delete gateway: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrGatewayNotFound
	}
	return nil
}

func (r *GatewayRepository) populateGatewayRelations(ctx context.Context, gateway *entity.Gateway) error {
	if gateway == nil {
		return nil
	}

	templateRows, err := r.db.Query(ctx, `
		SELECT template_id
		FROM smtp.gateway_templates
		WHERE gateway_id = $1 AND enabled = TRUE
		ORDER BY position ASC, template_id ASC
	`, gateway.ID)
	if err != nil {
		return fmt.Errorf("smtp repo: list gateway templates: %w", err)
	}
	for templateRows.Next() {
		var templateID string
		if err := templateRows.Scan(&templateID); err != nil {
			templateRows.Close()
			return fmt.Errorf("smtp repo: scan gateway template: %w", err)
		}
		gateway.TemplateIDs = append(gateway.TemplateIDs, templateID)
	}
	templateRows.Close()

	endpointRows, err := r.db.Query(ctx, `
		SELECT endpoint_id
		FROM smtp.gateway_endpoints
		WHERE gateway_id = $1 AND enabled = TRUE
		ORDER BY position ASC, endpoint_id ASC
	`, gateway.ID)
	if err != nil {
		return fmt.Errorf("smtp repo: list gateway endpoints: %w", err)
	}
	for endpointRows.Next() {
		var endpointID string
		if err := endpointRows.Scan(&endpointID); err != nil {
			endpointRows.Close()
			return fmt.Errorf("smtp repo: scan gateway endpoint: %w", err)
		}
		gateway.EndpointIDs = append(gateway.EndpointIDs, endpointID)
	}
	endpointRows.Close()

	return nil
}

func (r *GatewayRepository) getGatewayByQuery(ctx context.Context, query string, args ...any) (*entity.Gateway, error) {
	var row smtp_model.Gateway
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.ZoneID, &row.Name, &row.TrafficClass, &row.Status,
		&row.RoutingMode, &row.Priority, &row.FallbackGatewayID, &row.RuntimeVersion, &row.DesiredShardCount,
		&row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrGatewayNotFound
		}
		return nil, fmt.Errorf("smtp repo: get gateway: %w", err)
	}
	return smtp_model.GatewayModelToEntity(&row), nil
}

func scanGateway(rows pgx.Rows) (*entity.Gateway, error) {
	var row smtp_model.Gateway
	if err := rows.Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.ZoneID, &row.Name, &row.TrafficClass, &row.Status,
		&row.RoutingMode, &row.Priority, &row.FallbackGatewayID, &row.RuntimeVersion, &row.DesiredShardCount,
		&row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("smtp repo: scan gateway: %w", err)
	}
	return smtp_model.GatewayModelToEntity(&row), nil
}

func syncGatewayTemplates(ctx context.Context, tx pgx.Tx, gatewayID string, templateIDs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM smtp.gateway_templates WHERE gateway_id = $1`, gatewayID); err != nil {
		return fmt.Errorf("smtp repo: delete gateway templates: %w", err)
	}
	for idx, templateID := range templateIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO smtp.gateway_templates (gateway_id, template_id, position, enabled, created_at)
			VALUES ($1, $2, $3, TRUE, NOW())
		`, gatewayID, templateID, idx); err != nil {
			return fmt.Errorf("smtp repo: insert gateway template: %w", err)
		}
	}
	return nil
}

func syncGatewayEndpoints(ctx context.Context, tx pgx.Tx, gatewayID string, endpointIDs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM smtp.gateway_endpoints WHERE gateway_id = $1`, gatewayID); err != nil {
		return fmt.Errorf("smtp repo: delete gateway endpoints: %w", err)
	}
	for idx, endpointID := range endpointIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO smtp.gateway_endpoints (gateway_id, endpoint_id, position, enabled, created_at)
			VALUES ($1, $2, $3, TRUE, NOW())
		`, gatewayID, endpointID, idx); err != nil {
			return fmt.Errorf("smtp repo: insert gateway endpoint: %w", err)
		}
	}
	return nil
}

func syncGatewayShards(ctx context.Context, tx pgx.Tx, gatewayID string, desiredShardCount int) error {
	if _, err := tx.Exec(ctx, `DELETE FROM smtp.gateway_shards WHERE gateway_id = $1 AND shard_id >= $2`, gatewayID, desiredShardCount); err != nil {
		return fmt.Errorf("smtp repo: delete extra gateway shards: %w", err)
	}
	for shardID := 0; shardID < desiredShardCount; shardID++ {
		if _, err := tx.Exec(ctx, `
			INSERT INTO smtp.gateway_shards (gateway_id, shard_id, desired_state, created_at, updated_at)
			VALUES ($1, $2, 'active', NOW(), NOW())
			ON CONFLICT (gateway_id, shard_id) DO UPDATE SET
				desired_state = EXCLUDED.desired_state,
				updated_at = NOW()
		`, gatewayID, shardID); err != nil {
			return fmt.Errorf("smtp repo: upsert gateway shard: %w", err)
		}
	}
	return nil
}
