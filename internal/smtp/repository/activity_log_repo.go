package repository

import (
	"context"
	"fmt"

	"controlplane/internal/smtp/domain/entity"
	smtp_model "controlplane/internal/smtp/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityLogRepository struct {
	db *pgxpool.Pool
}

func NewActivityLogRepository(db *pgxpool.Pool) *ActivityLogRepository {
	return &ActivityLogRepository{db: db}
}

func (r *ActivityLogRepository) ListActivityLogs(ctx context.Context, workspaceID string) ([]*entity.ActivityLog, error) {
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
