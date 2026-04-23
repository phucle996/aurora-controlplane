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

type TemplateRepository struct {
	db *pgxpool.Pool
}

func NewTemplateRepository(db *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) ListTemplateItemsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.TemplateListItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			t.id,
			t.name,
			t.category,
			t.traffic_class,
			t.subject,
			t.from_email,
			t.to_email,
			t.status::text,
			COALESCE(t.consumer_id, ''),
			COALESCE(c.name, ''),
			t.updated_at
		FROM smtp.templates t
		LEFT JOIN smtp.consumers c ON c.id = t.consumer_id
		WHERE t.workspace_id = $1
		ORDER BY t.created_at DESC, t.id DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list template items: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.TemplateListItem, 0)
	for rows.Next() {
		var item entity.TemplateListItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Category,
			&item.TrafficClass,
			&item.Subject,
			&item.FromEmail,
			&item.ToEmail,
			&item.Status,
			&item.ConsumerID,
			&item.ConsumerName,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("smtp repo: scan template item: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate template items: %w", err)
	}
	return items, nil
}

func (r *TemplateRepository) GetTemplateDetail(ctx context.Context, workspaceID, templateID string) (*entity.TemplateDetail, error) {
	var item entity.TemplateDetail
	err := r.db.QueryRow(ctx, `
		SELECT
			t.id,
			t.name,
			t.category,
			t.traffic_class,
			t.subject,
			t.from_email,
			t.to_email,
			t.status::text,
			t.variables,
			COALESCE(t.consumer_id, ''),
			COALESCE(c.name, ''),
			t.text_body,
			t.html_body,
			t.active_version,
			t.runtime_version,
			t.created_at,
			t.updated_at
		FROM smtp.templates t
		LEFT JOIN smtp.consumers c ON c.id = t.consumer_id
		WHERE t.workspace_id = $1 AND t.id = $2
	`, workspaceID, templateID).Scan(
		&item.ID,
		&item.Name,
		&item.Category,
		&item.TrafficClass,
		&item.Subject,
		&item.FromEmail,
		&item.ToEmail,
		&item.Status,
		&item.Variables,
		&item.ConsumerID,
		&item.ConsumerName,
		&item.TextBody,
		&item.HTMLBody,
		&item.ActiveVersion,
		&item.RuntimeVersion,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("smtp repo: get template detail: %w", err)
	}
	return &item, nil
}

func (r *TemplateRepository) ListTemplatesByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Template, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, workspace_id, owner_user_id, name, category, traffic_class, subject, from_email, to_email,
			status::text, variables, consumer_id, active_version, retry_max_attempts, retry_backoff_seconds,
			text_body, html_body, runtime_version, created_at, updated_at
		FROM smtp.templates
		WHERE workspace_id = $1
		ORDER BY created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("smtp repo: list templates: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.Template, 0)
	for rows.Next() {
		row, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("smtp repo: iterate templates: %w", err)
	}
	return items, nil
}

func (r *TemplateRepository) GetTemplate(ctx context.Context, workspaceID, templateID string) (*entity.Template, error) {
	return r.getTemplateByQuery(ctx, `
		SELECT
			id, workspace_id, owner_user_id, name, category, traffic_class, subject, from_email, to_email,
			status::text, variables, consumer_id, active_version, retry_max_attempts, retry_backoff_seconds,
			text_body, html_body, runtime_version, created_at, updated_at
		FROM smtp.templates
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, templateID)
}

func (r *TemplateRepository) GetTemplateByID(ctx context.Context, templateID string) (*entity.Template, error) {
	return r.getTemplateByQuery(ctx, `
		SELECT
			id, workspace_id, owner_user_id, name, category, traffic_class, subject, from_email, to_email,
			status::text, variables, consumer_id, active_version, retry_max_attempts, retry_backoff_seconds,
			text_body, html_body, runtime_version, created_at, updated_at
		FROM smtp.templates
		WHERE id = $1
	`, templateID)
}

func (r *TemplateRepository) CreateTemplate(ctx context.Context, template *entity.Template) error {
	if template == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin create template tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.TemplateEntityToModel(template)
	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.templates (
			id, workspace_id, owner_user_id, name, category, traffic_class, subject, from_email, to_email,
			status, variables, consumer_id, active_version, retry_max_attempts, retry_backoff_seconds,
			text_body, html_body, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10::smtp.template_status, $11, $12, 1, $13, $14,
			$15, $16, NOW(), NOW()
		)
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.Name, row.Category, row.TrafficClass, row.Subject, row.FromEmail,
		row.ToEmail, row.Status, row.Variables, row.ConsumerID, row.RetryMaxAttempts, row.RetryBackoffSeconds,
		row.TextBody, row.HTMLBody,
	)
	if err != nil {
		return fmt.Errorf("smtp repo: create template: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.template_versions (
			template_id, version, subject, from_email, to_email, variables, text_body, html_body, created_at
		) VALUES ($1, 1, $2, $3, $4, $5, $6, $7, NOW())
	`, row.ID, row.Subject, row.FromEmail, row.ToEmail, row.Variables, row.TextBody, row.HTMLBody)
	if err != nil {
		return fmt.Errorf("smtp repo: create template version: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *TemplateRepository) UpdateTemplate(ctx context.Context, template *entity.Template) error {
	if template == nil {
		return smtp_errorx.ErrInvalidResource
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("smtp repo: begin update template tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := smtp_model.TemplateEntityToModel(template)
	var nextVersion int
	err = tx.QueryRow(ctx, `
		UPDATE smtp.templates
		SET
			workspace_id = $2,
			owner_user_id = $3,
			name = $4,
			category = $5,
			traffic_class = $6,
			subject = $7,
			from_email = $8,
			to_email = $9,
			status = $10::smtp.template_status,
			variables = $11,
			consumer_id = $12,
			active_version = active_version + 1,
			retry_max_attempts = $13,
			retry_backoff_seconds = $14,
			text_body = $15,
			html_body = $16,
			updated_at = NOW()
		WHERE id = $1
		RETURNING active_version
	`,
		row.ID, row.WorkspaceID, row.OwnerUserID, row.Name, row.Category, row.TrafficClass, row.Subject, row.FromEmail,
		row.ToEmail, row.Status, row.Variables, row.ConsumerID, row.RetryMaxAttempts, row.RetryBackoffSeconds,
		row.TextBody, row.HTMLBody,
	).Scan(&nextVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return smtp_errorx.ErrTemplateNotFound
		}
		return fmt.Errorf("smtp repo: update template: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO smtp.template_versions (
			template_id, version, subject, from_email, to_email, variables, text_body, html_body, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`,
		row.ID, nextVersion, row.Subject, row.FromEmail, row.ToEmail, row.Variables, row.TextBody, row.HTMLBody,
	)
	if err != nil {
		return fmt.Errorf("smtp repo: update template version: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *TemplateRepository) DeleteTemplate(ctx context.Context, workspaceID, templateID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM smtp.templates WHERE workspace_id = $1 AND id = $2`, workspaceID, templateID)
	if err != nil {
		return fmt.Errorf("smtp repo: delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return smtp_errorx.ErrTemplateNotFound
	}
	return nil
}

func (r *TemplateRepository) getTemplateByQuery(ctx context.Context, query string, args ...any) (*entity.Template, error) {
	var row smtp_model.Template
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.Name, &row.Category, &row.TrafficClass,
		&row.Subject, &row.FromEmail, &row.ToEmail, &row.Status, &row.Variables, &row.ConsumerID,
		&row.ActiveVersion, &row.RetryMaxAttempts, &row.RetryBackoffSeconds, &row.TextBody, &row.HTMLBody,
		&row.RuntimeVersion, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, smtp_errorx.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("smtp repo: get template: %w", err)
	}
	return smtp_model.TemplateModelToEntity(&row), nil
}

func scanTemplate(rows pgx.Rows) (*entity.Template, error) {
	var row smtp_model.Template
	if err := rows.Scan(
		&row.ID, &row.WorkspaceID, &row.OwnerUserID, &row.Name, &row.Category, &row.TrafficClass,
		&row.Subject, &row.FromEmail, &row.ToEmail, &row.Status, &row.Variables, &row.ConsumerID,
		&row.ActiveVersion, &row.RetryMaxAttempts, &row.RetryBackoffSeconds, &row.TextBody, &row.HTMLBody,
		&row.RuntimeVersion, &row.CreatedAt, &row.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("smtp repo: scan template: %w", err)
	}
	return smtp_model.TemplateModelToEntity(&row), nil
}
