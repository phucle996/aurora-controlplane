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

type ConsumerRepository struct {
	db *pgxpool.Pool
}

func NewConsumerRepository(db *pgxpool.Pool) *ConsumerRepository {
	return &ConsumerRepository{db: db}
}

func (r *ConsumerRepository) ListConsumerViewsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.ConsumerView, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			c.id,
			COALESCE(c.zone_id, ''),
			c.name,
			c.transport_type::text,
			c.source,
			c.consumer_group,
			c.worker_concurrency,
			c.ack_timeout_seconds,
			c.batch_size,
			c.status::text,
			c.note,
			c.connection_config,
			c.desired_shard_count,
			(cs.consumer_id IS NOT NULL) AS has_secret,
			c.created_at,
			c.updated_at
		FROM smtp.consumers c
		LEFT JOIN smtp.consumer_secrets cs ON cs.consumer_id = c.id
		WHERE c.workspace_id = $1
		ORDER BY c.created_at DESC, c.id DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumer views: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.ConsumerView, 0)
	for rows.Next() {
		var item entity.ConsumerView
		if err := rows.Scan(
			&item.ID,
			&item.ZoneID,
			&item.Name,
			&item.TransportType,
			&item.Source,
			&item.ConsumerGroup,
			&item.WorkerConcurrency,
			&item.AckTimeoutSeconds,
			&item.BatchSize,
			&item.Status,
			&item.Note,
			&item.ConnectionConfig,
			&item.DesiredShardCount,
			&item.HasSecret,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan consumer view: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate consumer views: %w", err)
	}

	return items, nil
}

func (r *ConsumerRepository) GetConsumerView(ctx context.Context, workspaceID, consumerID string) (*entity.ConsumerView, error) {
	var item entity.ConsumerView
	err := r.db.QueryRow(ctx, `
		SELECT
			c.id,
			COALESCE(c.zone_id, ''),
			c.name,
			c.transport_type::text,
			c.source,
			c.consumer_group,
			c.worker_concurrency,
			c.ack_timeout_seconds,
			c.batch_size,
			c.status::text,
			c.note,
			c.connection_config,
			c.desired_shard_count,
			(cs.consumer_id IS NOT NULL) AS has_secret,
			c.created_at,
			c.updated_at
		FROM smtp.consumers c
		LEFT JOIN smtp.consumer_secrets cs ON cs.consumer_id = c.id
		WHERE c.workspace_id = $1 AND c.id = $2
	`, workspaceID, consumerID).Scan(
		&item.ID,
		&item.ZoneID,
		&item.Name,
		&item.TransportType,
		&item.Source,
		&item.ConsumerGroup,
		&item.WorkerConcurrency,
		&item.AckTimeoutSeconds,
		&item.BatchSize,
		&item.Status,
		&item.Note,
		&item.ConnectionConfig,
		&item.DesiredShardCount,
		&item.HasSecret,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrConsumerNotFound
		}
		return nil, fmt.Errorf("smtp repo: get consumer view: %w", err)
	}
	return &item, nil
}

func (r *ConsumerRepository) ListConsumerOptionsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.ConsumerOption, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, status::text
		FROM smtp.consumers
		WHERE workspace_id = $1
		ORDER BY name ASC, created_at ASC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumer options: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.ConsumerOption, 0)
	for rows.Next() {
		var item entity.ConsumerOption
		if err := rows.Scan(&item.ID, &item.Label, &item.Status); err != nil {
			return nil, fmt.Errorf("smtp repo: scan consumer option: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate consumer options: %w", err)
	}
	return items, nil
}

func (r *ConsumerRepository) ListConsumersByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Consumer, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			c.id,
			c.workspace_id,
			c.owner_user_id,
			c.zone_id,
			c.name,
			c.transport_type::text,
			c.source,
			c.consumer_group,
			c.worker_concurrency,
			c.ack_timeout_seconds,
			c.batch_size,
			c.status::text,
			c.note,
			c.connection_config,
			c.runtime_version,
			c.desired_shard_count,
			COALESCE(cs.secret_config, '{}'::jsonb),
			COALESCE(cs.secret_ref, ''),
			COALESCE(cs.secret_version, 1),
			COALESCE(cs.provider, 'postgresql'),
			c.created_at,
			c.updated_at
		FROM smtp.consumers c
		LEFT JOIN smtp.consumer_secrets cs ON cs.consumer_id = c.id
		WHERE c.workspace_id = $1
		ORDER BY c.created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list consumers: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.Consumer, 0)
	for rows.Next() {
		row, err := scanConsumer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate consumers: %w", err)
	}

	return items, nil
}

func (r *ConsumerRepository) GetConsumer(ctx context.Context, workspaceID, consumerID string) (*entity.Consumer, error) {
	row, err := r.getConsumerByQuery(ctx, `
		SELECT
			c.id,
			c.workspace_id,
			c.owner_user_id,
			c.zone_id,
			c.name,
			c.transport_type::text,
			c.source,
			c.consumer_group,
			c.worker_concurrency,
			c.ack_timeout_seconds,
			c.batch_size,
			c.status::text,
			c.note,
			c.connection_config,
			c.runtime_version,
			c.desired_shard_count,
			COALESCE(cs.secret_config, '{}'::jsonb),
			COALESCE(cs.secret_ref, ''),
			COALESCE(cs.secret_version, 1),
			COALESCE(cs.provider, 'postgresql'),
			c.created_at,
			c.updated_at
		FROM smtp.consumers c
		LEFT JOIN smtp.consumer_secrets cs ON cs.consumer_id = c.id
		WHERE c.workspace_id = $1 AND c.id = $2
	`, workspaceID, consumerID)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *ConsumerRepository) GetConsumerByID(ctx context.Context, consumerID string) (*entity.Consumer, error) {
	return r.getConsumerByQuery(ctx, `
		SELECT
			c.id,
			c.workspace_id,
			c.owner_user_id,
			c.zone_id,
			c.name,
			c.transport_type::text,
			c.source,
			c.consumer_group,
			c.worker_concurrency,
			c.ack_timeout_seconds,
			c.batch_size,
			c.status::text,
			c.note,
			c.connection_config,
			c.runtime_version,
			c.desired_shard_count,
			COALESCE(cs.secret_config, '{}'::jsonb),
			COALESCE(cs.secret_ref, ''),
			COALESCE(cs.secret_version, 1),
			COALESCE(cs.provider, 'postgresql'),
			c.created_at,
			c.updated_at
		FROM smtp.consumers c
		LEFT JOIN smtp.consumer_secrets cs ON cs.consumer_id = c.id
		WHERE c.id = $1
	`, consumerID)
}

func (r *ConsumerRepository) CreateConsumer(ctx context.Context, consumer *entity.Consumer) error {
	if consumer == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin create consumer tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.ConsumerEntityToModel(consumer)
	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.consumers (
			id, workspace_id, owner_user_id, zone_id, name, transport_type, source, consumer_group,
			worker_concurrency, ack_timeout_seconds, batch_size, status, note, connection_config,
			desired_shard_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6::smtp.consumer_transport_type, $7, $8,
			$9, $10, $11, $12::smtp.consumer_status, $13, $14::jsonb,
			$15, NOW(), NOW()
		)
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.ZoneID, row.Name, row.TransportType, row.Source, row.ConsumerGroup,
		row.WorkerConcurrency, row.AckTimeoutSeconds, row.BatchSize, row.Status, row.Note, row.ConnectionConfig,
		maxInt(row.DesiredShardCount, 1),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: create consumer: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.consumer_secrets (
			consumer_id, secret_config, secret_ref, secret_version, provider, updated_at
		) VALUES ($1, $2::jsonb, $3, 1, $4, NOW())
	`,
		row.ID, row.SecretConfig, row.SecretRef, defaultString(row.SecretProvider, "postgresql"),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: create consumer secrets: %w", err)
	}

	if err := syncConsumerShards(ctx, tx, row.ID, maxInt(row.DesiredShardCount, 1)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *ConsumerRepository) UpdateConsumer(ctx context.Context, consumer *entity.Consumer) error {
	if consumer == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin update consumer tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.ConsumerEntityToModel(consumer)
	tag, err := tx.Exec(ctx, `
		UPDATE smtp.consumers
		SET
			workspace_id = $2,
			owner_user_id = $3,
			zone_id = $4,
			name = $5,
			transport_type = $6::smtp.consumer_transport_type,
			source = $7,
			consumer_group = $8,
			worker_concurrency = $9,
			ack_timeout_seconds = $10,
			batch_size = $11,
			status = $12::smtp.consumer_status,
			note = $13,
			connection_config = $14::jsonb,
			desired_shard_count = $15,
			updated_at = NOW()
		WHERE id = $1
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.ZoneID, row.Name, row.TransportType, row.Source, row.ConsumerGroup,
		row.WorkerConcurrency, row.AckTimeoutSeconds, row.BatchSize, row.Status, row.Note, row.ConnectionConfig,
		maxInt(row.DesiredShardCount, 1),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: update consumer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrConsumerNotFound
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.consumer_secrets (
			consumer_id, secret_config, secret_ref, secret_version, provider, updated_at
		) VALUES ($1, $2::jsonb, $3, 1, $4, NOW())
		ON CONFLICT (consumer_id) DO UPDATE SET
			secret_config = EXCLUDED.secret_config,
			secret_ref = EXCLUDED.secret_ref,
			secret_version = smtp.consumer_secrets.secret_version + 1,
			provider = EXCLUDED.provider,
			updated_at = NOW()
	`,
		row.ID, row.SecretConfig, row.SecretRef, defaultString(row.SecretProvider, "postgresql"),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: update consumer secrets: %w", err)
	}

	if err := syncConsumerShards(ctx, tx, row.ID, maxInt(row.DesiredShardCount, 1)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *ConsumerRepository) DeleteConsumer(ctx context.Context, workspaceID, consumerID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM smtp.consumers WHERE workspace_id = $1 AND id = $2`, workspaceID, consumerID)
	if err != nil {
		return fmt.Errorf("smtp repo: delete consumer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrConsumerNotFound
	}
	return nil
}

func (r *ConsumerRepository) getConsumerByQuery(ctx context.Context, query string, args ...any) (*entity.Consumer, error) {
	var row smtp_model.Consumer
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.ZoneID, &row.Name, &row.TransportType, &row.Source,
		&row.ConsumerGroup, &row.WorkerConcurrency, &row.AckTimeoutSeconds, &row.BatchSize, &row.Status,
		&row.Note, &row.ConnectionConfig, &row.RuntimeVersion, &row.DesiredShardCount, &row.SecretConfig,
		&row.SecretRef, &row.SecretVersion, &row.SecretProvider, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrConsumerNotFound
		}
		return nil, fmt.Errorf("smtp repo: get consumer: %w", err)
	}
	return smtp_model.ConsumerModelToEntity(&row), nil
}

func scanConsumer(rows pgx.Rows) (*entity.Consumer, error) {
	var row smtp_model.Consumer
	if err := rows.Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.ZoneID, &row.Name, &row.TransportType, &row.Source,
		&row.ConsumerGroup, &row.WorkerConcurrency, &row.AckTimeoutSeconds, &row.BatchSize, &row.Status,
		&row.Note, &row.ConnectionConfig, &row.RuntimeVersion, &row.DesiredShardCount, &row.SecretConfig,
		&row.SecretRef, &row.SecretVersion, &row.SecretProvider, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("smtp repo: scan consumer: %w", err)
	}
	return smtp_model.ConsumerModelToEntity(&row), nil
}

func syncConsumerShards(ctx context.Context, tx pgx.Tx, consumerID string, desiredShardCount int) error {
	if _, err := tx.Exec(ctx, `DELETE FROM smtp.consumer_shards WHERE consumer_id = $1 AND shard_id >= $2`, consumerID, desiredShardCount); err != nil {
		return fmt.Errorf("smtp repo: delete extra consumer shards: %w", err)
	}
	for shardID := 0; shardID < desiredShardCount; shardID++ {
		if _, err := tx.Exec(ctx, `
			INSERT INTO smtp.consumer_shards (consumer_id, shard_id, desired_state, created_at, updated_at)
			VALUES ($1, $2, 'active', NOW(), NOW())
			ON CONFLICT (consumer_id, shard_id) DO UPDATE SET
				desired_state = EXCLUDED.desired_state,
				updated_at = NOW()
		`, consumerID, shardID); err != nil {
			return fmt.Errorf("smtp repo: upsert consumer shard: %w", err)
		}
	}
	return nil
}
