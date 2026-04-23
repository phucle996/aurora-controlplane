package repository

import (
	"context"
	"fmt"

	"controlplane/internal/smtp/domain/entity"
	smtp_model "controlplane/internal/smtp/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliveryAttemptRepository struct {
	db *pgxpool.Pool
}

func NewDeliveryAttemptRepository(db *pgxpool.Pool) *DeliveryAttemptRepository {
	return &DeliveryAttemptRepository{db: db}
}

func (r *DeliveryAttemptRepository) ListDeliveryAttempts(ctx context.Context, workspaceID string) ([]*entity.DeliveryAttempt, error) {
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
