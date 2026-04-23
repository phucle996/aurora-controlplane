package smtp_domainsvc

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type TemplateService interface {
	ListTemplateItems(ctx context.Context, workspaceID string) ([]*entity.TemplateListItem, error)
	GetTemplateDetail(ctx context.Context, workspaceID, templateID string) (*entity.TemplateDetail, error)
	ListTemplates(ctx context.Context, workspaceID string) ([]*entity.Template, error)
	GetTemplate(ctx context.Context, workspaceID, templateID string) (*entity.Template, error)
	CreateTemplate(ctx context.Context, template *entity.Template) error
	UpdateTemplate(ctx context.Context, template *entity.Template) error
	DeleteTemplate(ctx context.Context, workspaceID, templateID string) error
}
