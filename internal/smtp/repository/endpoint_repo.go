package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"controlplane/internal/security"
	"controlplane/internal/smtp/domain/entity"
	smtp_errorx "controlplane/internal/smtp/errorx"
	smtp_model "controlplane/internal/smtp/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EndpointRepository struct {
	db        *pgxpool.Pool
	masterKey string
}

func NewEndpointRepository(db *pgxpool.Pool, masterKey string) *EndpointRepository {
	return &EndpointRepository{db: db, masterKey: masterKey}
}

func (r *EndpointRepository) ListEndpointViewsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.EndpointView, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			e.id,
			e.name,
			e.provider_kind,
			e.host,
			e.port,
			e.username,
			e.priority,
			e.weight,
			e.max_connections,
			e.max_parallel_sends,
			e.max_messages_per_second,
			e.burst,
			e.warmup_state,
			e.status::text,
			e.tls_mode::text,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.password, '') <> '') AS has_secret,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.ca_cert_pem, '') <> '') AS has_ca_cert,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.client_cert_pem, '') <> '') AS has_client_cert,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.client_key_pem, '') <> '') AS has_client_key,
			e.created_at,
			e.updated_at
		FROM smtp.endpoints e
		LEFT JOIN smtp.endpoint_secrets es ON es.endpoint_id = e.id
		WHERE e.workspace_id = $1
		ORDER BY e.created_at DESC, e.id DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list endpoint views: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.EndpointView, 0)
	for rows.Next() {
		var item entity.EndpointView
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.ProviderKind,
			&item.Host,
			&item.Port,
			&item.Username,
			&item.Priority,
			&item.Weight,
			&item.MaxConnections,
			&item.MaxParallelSends,
			&item.MaxMessagesPerSecond,
			&item.Burst,
			&item.WarmupState,
			&item.Status,
			&item.TLSMode,
			&item.HasSecret,
			&item.HasCACert,
			&item.HasClientCert,
			&item.HasClientKey,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan endpoint view: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate endpoint views: %w", err)
	}
	return items, nil
}

func (r *EndpointRepository) GetEndpointView(ctx context.Context, workspaceID, endpointID string) (*entity.EndpointView, error) {
	var item entity.EndpointView
	err := r.db.QueryRow(ctx, `
		SELECT
			e.id,
			e.name,
			e.provider_kind,
			e.host,
			e.port,
			e.username,
			e.priority,
			e.weight,
			e.max_connections,
			e.max_parallel_sends,
			e.max_messages_per_second,
			e.burst,
			e.warmup_state,
			e.status::text,
			e.tls_mode::text,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.password, '') <> '') AS has_secret,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.ca_cert_pem, '') <> '') AS has_ca_cert,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.client_cert_pem, '') <> '') AS has_client_cert,
			(es.endpoint_id IS NOT NULL AND COALESCE(es.client_key_pem, '') <> '') AS has_client_key,
			e.created_at,
			e.updated_at
		FROM smtp.endpoints e
		LEFT JOIN smtp.endpoint_secrets es ON es.endpoint_id = e.id
		WHERE e.workspace_id = $1 AND e.id = $2
	`, workspaceID, endpointID).Scan(
		&item.ID,
		&item.Name,
		&item.ProviderKind,
		&item.Host,
		&item.Port,
		&item.Username,
		&item.Priority,
		&item.Weight,
		&item.MaxConnections,
		&item.MaxParallelSends,
		&item.MaxMessagesPerSecond,
		&item.Burst,
		&item.WarmupState,
		&item.Status,
		&item.TLSMode,
		&item.HasSecret,
		&item.HasCACert,
		&item.HasClientCert,
		&item.HasClientKey,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrEndpointNotFound
		}
		return nil, fmt.Errorf("smtp repo: get endpoint view: %w", err)
	}
	return &item, nil
}

func (r *EndpointRepository) ListEndpointsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Endpoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			e.id, e.workspace_id, e.owner_user_id, e.name, e.provider_kind, e.host, e.port, e.username,
			e.priority, e.weight, e.max_connections, e.max_parallel_sends, e.max_messages_per_second,
			e.burst, e.warmup_state, e.status::text, e.tls_mode::text, e.runtime_version,
			COALESCE(es.password, ''), COALESCE(es.ca_cert_pem, ''), COALESCE(es.client_cert_pem, ''),
			COALESCE(es.client_key_pem, ''), COALESCE(es.secret_ref, ''), COALESCE(es.secret_version, 1),
			COALESCE(es.provider, 'postgresql'),
			e.created_at, e.updated_at
		FROM smtp.endpoints e
		LEFT JOIN smtp.endpoint_secrets es ON es.endpoint_id = e.id
		WHERE e.workspace_id = $1
		ORDER BY e.created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list endpoints: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.Endpoint, 0)
	for rows.Next() {
		row, err := scanEndpoint(rows, r.masterKey)
		if err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate endpoints: %w", err)
	}
	return items, nil
}

func (r *EndpointRepository) GetEndpoint(ctx context.Context, workspaceID, endpointID string) (*entity.Endpoint, error) {
	return r.getEndpointByQuery(ctx, `
		SELECT
			e.id, e.workspace_id, e.owner_user_id, e.name, e.provider_kind, e.host, e.port, e.username,
			e.priority, e.weight, e.max_connections, e.max_parallel_sends, e.max_messages_per_second,
			e.burst, e.warmup_state, e.status::text, e.tls_mode::text, e.runtime_version,
			COALESCE(es.password, ''), COALESCE(es.ca_cert_pem, ''), COALESCE(es.client_cert_pem, ''),
			COALESCE(es.client_key_pem, ''), COALESCE(es.secret_ref, ''), COALESCE(es.secret_version, 1),
			COALESCE(es.provider, 'postgresql'),
			e.created_at, e.updated_at
		FROM smtp.endpoints e
		LEFT JOIN smtp.endpoint_secrets es ON es.endpoint_id = e.id
		WHERE e.workspace_id = $1 AND e.id = $2
	`, workspaceID, endpointID)
}

func (r *EndpointRepository) GetEndpointByID(ctx context.Context, endpointID string) (*entity.Endpoint, error) {
	return r.getEndpointByQuery(ctx, `
		SELECT
			e.id, e.workspace_id, e.owner_user_id, e.name, e.provider_kind, e.host, e.port, e.username,
			e.priority, e.weight, e.max_connections, e.max_parallel_sends, e.max_messages_per_second,
			e.burst, e.warmup_state, e.status::text, e.tls_mode::text, e.runtime_version,
			COALESCE(es.password, ''), COALESCE(es.ca_cert_pem, ''), COALESCE(es.client_cert_pem, ''),
			COALESCE(es.client_key_pem, ''), COALESCE(es.secret_ref, ''), COALESCE(es.secret_version, 1),
			COALESCE(es.provider, 'postgresql'),
			e.created_at, e.updated_at
		FROM smtp.endpoints e
		LEFT JOIN smtp.endpoint_secrets es ON es.endpoint_id = e.id
		WHERE e.id = $1
	`, endpointID)
}

func (r *EndpointRepository) CreateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error {
	if endpoint == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin create endpoint tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.EndpointEntityToModel(endpoint)
	row.Password, err = encodeEndpointSecretValue(r.masterKey, row.Password)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint password: %w", err)
	}
	row.CACertPEM, err = encodeEndpointSecretValue(r.masterKey, row.CACertPEM)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint ca cert: %w", err)
	}
	row.ClientCertPEM, err = encodeEndpointSecretValue(r.masterKey, row.ClientCertPEM)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint client cert: %w", err)
	}
	row.ClientKeyPEM, err = encodeEndpointSecretValue(r.masterKey, row.ClientKeyPEM)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint client key: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.endpoints (
			id, workspace_id, owner_user_id, name, provider_kind, host, port, username, priority, weight,
			max_connections, max_parallel_sends, max_messages_per_second, burst, warmup_state, status,
			tls_mode, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16::smtp.endpoint_status,
			$17::smtp.tls_mode, NOW(), NOW()
		)
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.Name, row.ProviderKind, row.Host, row.Port, row.Username,
		row.Priority, row.Weight, row.MaxConnections, row.MaxParallelSends, row.MaxMessagesPerSecond, row.Burst,
		row.WarmupState, row.Status, row.TLSMode,
	)
	if err != nil {
		return fmt.Errorf("smtp repo: create endpoint: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.endpoint_secrets (
			endpoint_id, password, ca_cert_pem, client_cert_pem, client_key_pem, secret_ref, secret_version, provider, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 1, $7, NOW())
	`,
		row.ID, row.Password, row.CACertPEM, row.ClientCertPEM, row.ClientKeyPEM, row.SecretRef, defaultString(row.SecretProvider, "postgresql"),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: create endpoint secrets: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *EndpointRepository) UpdateEndpoint(ctx context.Context, endpoint *entity.Endpoint) error {
	if endpoint == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin update endpoint tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.EndpointEntityToModel(endpoint)
	row.Password, err = encodeEndpointSecretValue(r.masterKey, row.Password)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint password: %w", err)
	}
	row.CACertPEM, err = encodeEndpointSecretValue(r.masterKey, row.CACertPEM)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint ca cert: %w", err)
	}
	row.ClientCertPEM, err = encodeEndpointSecretValue(r.masterKey, row.ClientCertPEM)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint client cert: %w", err)
	}
	row.ClientKeyPEM, err = encodeEndpointSecretValue(r.masterKey, row.ClientKeyPEM)
	if err != nil {
		return fmt.Errorf("smtp repo: encrypt endpoint client key: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE smtp.endpoints
		SET
			workspace_id = $2,
			owner_user_id = $3,
			name = $4,
			provider_kind = $5,
			host = $6,
			port = $7,
			username = $8,
			priority = $9,
			weight = $10,
			max_connections = $11,
			max_parallel_sends = $12,
			max_messages_per_second = $13,
			burst = $14,
			warmup_state = $15,
			status = $16::smtp.endpoint_status,
			tls_mode = $17::smtp.tls_mode,
			updated_at = NOW()
		WHERE id = $1
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.Name, row.ProviderKind, row.Host, row.Port, row.Username,
		row.Priority, row.Weight, row.MaxConnections, row.MaxParallelSends, row.MaxMessagesPerSecond, row.Burst,
		row.WarmupState, row.Status, row.TLSMode,
	)
	if err != nil {
		return fmt.Errorf("smtp repo: update endpoint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrEndpointNotFound
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.endpoint_secrets (
			endpoint_id, password, ca_cert_pem, client_cert_pem, client_key_pem, secret_ref, secret_version, provider, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 1, $7, NOW())
		ON CONFLICT (endpoint_id) DO UPDATE SET
			password = EXCLUDED.password,
			ca_cert_pem = EXCLUDED.ca_cert_pem,
			client_cert_pem = EXCLUDED.client_cert_pem,
			client_key_pem = EXCLUDED.client_key_pem,
			secret_ref = EXCLUDED.secret_ref,
			secret_version = smtp.endpoint_secrets.secret_version + 1,
			provider = EXCLUDED.provider,
			updated_at = NOW()
	`,
		row.ID, row.Password, row.CACertPEM, row.ClientCertPEM, row.ClientKeyPEM, row.SecretRef, defaultString(row.SecretProvider, "postgresql"),
	)
	if err != nil {
		return fmt.Errorf("smtp repo: update endpoint secrets: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *EndpointRepository) DeleteEndpoint(ctx context.Context, workspaceID, endpointID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM smtp.endpoints WHERE workspace_id = $1 AND id = $2`, workspaceID, endpointID)
	if err != nil {
		return fmt.Errorf("smtp repo: delete endpoint: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrEndpointNotFound
	}
	return nil
}

func (r *EndpointRepository) getEndpointByQuery(ctx context.Context, query string, args ...any) (*entity.Endpoint, error) {
	var row smtp_model.Endpoint
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.Name, &row.ProviderKind, &row.Host, &row.Port,
		&row.Username, &row.Priority, &row.Weight, &row.MaxConnections, &row.MaxParallelSends,
		&row.MaxMessagesPerSecond, &row.Burst, &row.WarmupState, &row.Status, &row.TLSMode, &row.RuntimeVersion,
		&row.Password, &row.CACertPEM, &row.ClientCertPEM, &row.ClientKeyPEM, &row.SecretRef, &row.SecretVersion,
		&row.SecretProvider, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrEndpointNotFound
		}
		return nil, fmt.Errorf("smtp repo: get endpoint: %w", err)
	}
	if err := decodeEndpointModelSecrets(r.masterKey, &row); err != nil {
		return nil, err
	}
	return smtp_model.EndpointModelToEntity(&row), nil
}

func scanEndpoint(rows pgx.Rows, masterKey string) (*entity.Endpoint, error) {
	var row smtp_model.Endpoint
	if err := rows.Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.Name, &row.ProviderKind, &row.Host, &row.Port,
		&row.Username, &row.Priority, &row.Weight, &row.MaxConnections, &row.MaxParallelSends,
		&row.MaxMessagesPerSecond, &row.Burst, &row.WarmupState, &row.Status, &row.TLSMode, &row.RuntimeVersion,
		&row.Password, &row.CACertPEM, &row.ClientCertPEM, &row.ClientKeyPEM, &row.SecretRef, &row.SecretVersion,
		&row.SecretProvider, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("smtp repo: scan endpoint: %w", err)
	}
	if err := decodeEndpointModelSecrets(masterKey, &row); err != nil {
		return nil, err
	}
	return smtp_model.EndpointModelToEntity(&row), nil
}

const endpointSecretCipherPrefix = "enc:"

func encodeEndpointSecretValue(masterKey, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	encrypted, err := security.EncryptSecret(value, masterKey)
	if err != nil {
		return "", err
	}
	return endpointSecretCipherPrefix + encrypted, nil
}

func decodeEndpointSecretValue(masterKey, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, endpointSecretCipherPrefix) {
		return value, nil
	}

	return security.DecryptSecret(strings.TrimPrefix(value, endpointSecretCipherPrefix), masterKey)
}

func decodeEndpointModelSecrets(masterKey string, row *smtp_model.Endpoint) error {
	var err error
	if row.Password, err = decodeEndpointSecretValue(masterKey, row.Password); err != nil {
		return fmt.Errorf("smtp repo: decrypt endpoint password: %w", err)
	}
	if row.CACertPEM, err = decodeEndpointSecretValue(masterKey, row.CACertPEM); err != nil {
		return fmt.Errorf("smtp repo: decrypt endpoint ca cert: %w", err)
	}
	if row.ClientCertPEM, err = decodeEndpointSecretValue(masterKey, row.ClientCertPEM); err != nil {
		return fmt.Errorf("smtp repo: decrypt endpoint client cert: %w", err)
	}
	if row.ClientKeyPEM, err = decodeEndpointSecretValue(masterKey, row.ClientKeyPEM); err != nil {
		return fmt.Errorf("smtp repo: decrypt endpoint client key: %w", err)
	}
	return nil
}
