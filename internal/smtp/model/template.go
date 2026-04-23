package smtp_model

import (
	"time"

	"controlplane/internal/smtp/domain/entity"
)

type Template struct {
	ID                  string    `db:"id"`
	WorkspaceID         *string   `db:"workspace_id"`
	OwnerUserID         *string   `db:"owner_user_id"`
	Name                string    `db:"name"`
	Category            string    `db:"category"`
	TrafficClass        string    `db:"traffic_class"`
	Subject             string    `db:"subject"`
	FromEmail           string    `db:"from_email"`
	ToEmail             string    `db:"to_email"`
	Status              string    `db:"status"`
	Variables           []string  `db:"variables"`
	ConsumerID          *string   `db:"consumer_id"`
	ActiveVersion       int       `db:"active_version"`
	RetryMaxAttempts    int       `db:"retry_max_attempts"`
	RetryBackoffSeconds int       `db:"retry_backoff_seconds"`
	TextBody            string    `db:"text_body"`
	HTMLBody            string    `db:"html_body"`
	RuntimeVersion      int64     `db:"runtime_version"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

func TemplateEntityToModel(v *entity.Template) *Template {
	if v == nil {
		return nil
	}

	return &Template{
		ID:                  v.ID,
		WorkspaceID:         stringPtr(v.WorkspaceID),
		OwnerUserID:         stringPtr(v.OwnerUserID),
		Name:                v.Name,
		Category:            v.Category,
		TrafficClass:        v.TrafficClass,
		Subject:             v.Subject,
		FromEmail:           v.FromEmail,
		ToEmail:             v.ToEmail,
		Status:              v.Status,
		Variables:           v.Variables,
		ConsumerID:          stringPtr(v.ConsumerID),
		ActiveVersion:       v.ActiveVersion,
		RetryMaxAttempts:    v.RetryMaxAttempts,
		RetryBackoffSeconds: v.RetryBackoffSeconds,
		TextBody:            v.TextBody,
		HTMLBody:            v.HTMLBody,
		RuntimeVersion:      v.RuntimeVersion,
		CreatedAt:           v.CreatedAt,
		UpdatedAt:           v.UpdatedAt,
	}
}

func TemplateModelToEntity(v *Template) *entity.Template {
	if v == nil {
		return nil
	}

	return &entity.Template{
		ID:                  v.ID,
		WorkspaceID:         stringValue(v.WorkspaceID),
		OwnerUserID:         stringValue(v.OwnerUserID),
		Name:                v.Name,
		Category:            v.Category,
		TrafficClass:        v.TrafficClass,
		Subject:             v.Subject,
		FromEmail:           v.FromEmail,
		ToEmail:             v.ToEmail,
		Status:              v.Status,
		Variables:           v.Variables,
		ConsumerID:          stringValue(v.ConsumerID),
		ActiveVersion:       v.ActiveVersion,
		RetryMaxAttempts:    v.RetryMaxAttempts,
		RetryBackoffSeconds: v.RetryBackoffSeconds,
		TextBody:            v.TextBody,
		HTMLBody:            v.HTMLBody,
		RuntimeVersion:      v.RuntimeVersion,
		CreatedAt:           v.CreatedAt,
		UpdatedAt:           v.UpdatedAt,
	}
}
