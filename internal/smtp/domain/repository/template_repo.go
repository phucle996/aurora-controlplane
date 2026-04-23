package smtp_domainrepo

import (
	"context"

	"controlplane/internal/smtp/domain/entity"
)

type TemplateRepository interface {
	ListTemplateItemsByWorkspace(ctx context.Context, workspaceID string) ([]*entity.TemplateListItem, error)
	GetTemplateDetail(ctx context.Context, workspaceID, templateID string) (*entity.TemplateDetail, error)
	ListTemplatesByWorkspace(ctx context.Context, workspaceID string) ([]*entity.Template, error)
	GetTemplate(ctx context.Context, workspaceID, templateID string) (*entity.Template, error)
	GetTemplateByID(ctx context.Context, templateID string) (*entity.Template, error)
	CreateTemplate(ctx context.Context, template *entity.Template) error
	UpdateTemplate(ctx context.Context, template *entity.Template) error
	DeleteTemplate(ctx context.Context, workspaceID, templateID string) error
}
