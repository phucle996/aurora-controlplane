package service

import (
	"context"
	"strings"

	"controlplane/internal/smtp/domain/entity"
	smtp_domainrepo "controlplane/internal/smtp/domain/repository"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
	"controlplane/pkg/id"
)

type TemplateService struct {
	repo         smtp_domainrepo.TemplateRepository
	consumerRepo smtp_domainrepo.ConsumerRepository
}

func NewTemplateService(repo smtp_domainrepo.TemplateRepository,
	consumerRepo smtp_domainrepo.ConsumerRepository,
) smtp_domainsvc.TemplateService {
	return &TemplateService{
		repo:         repo,
		consumerRepo: consumerRepo,
	}
}

func (s *TemplateService) ListTemplateItems(ctx context.Context, workspaceID string) ([]*entity.TemplateListItem, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListTemplateItemsByWorkspace(ctx, workspaceID)
}

func (s *TemplateService) GetTemplateDetail(ctx context.Context, workspaceID, templateID string) (*entity.TemplateDetail, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetTemplateDetail(ctx, workspaceID, strings.TrimSpace(templateID))
}

func (s *TemplateService) ListTemplates(ctx context.Context, workspaceID string) ([]*entity.Template, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.ListTemplatesByWorkspace(ctx, workspaceID)
}

func (s *TemplateService) GetTemplate(ctx context.Context, workspaceID, templateID string) (*entity.Template, error) {
	if s == nil || s.repo == nil {
		return nil, smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.GetTemplate(ctx, workspaceID, strings.TrimSpace(templateID))
}

func (s *TemplateService) CreateTemplate(ctx context.Context, template *entity.Template) error {
	if s == nil || s.repo == nil || template == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if err := s.normalizeTemplate(ctx, template); err != nil {
		return err
	}
	templateID, err := id.Generate()
	if err != nil {
		return err
	}
	template.ID = templateID
	return s.repo.CreateTemplate(ctx, template)
}

func (s *TemplateService) UpdateTemplate(ctx context.Context, template *entity.Template) error {
	if s == nil || s.repo == nil || template == nil {
		return smtp_errorx.ErrInvalidResource
	}
	if strings.TrimSpace(template.ID) == "" {
		return smtp_errorx.ErrInvalidResource
	}
	if err := s.normalizeTemplate(ctx, template); err != nil {
		return err
	}
	return s.repo.UpdateTemplate(ctx, template)
}

func (s *TemplateService) DeleteTemplate(ctx context.Context, workspaceID, templateID string) error {
	if s == nil || s.repo == nil {
		return smtp_errorx.ErrInvalidResource
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	return s.repo.DeleteTemplate(ctx, workspaceID, strings.TrimSpace(templateID))
}

func (s *TemplateService) normalizeTemplate(ctx context.Context, template *entity.Template) error {
	template.WorkspaceID = trimString(template.WorkspaceID)
	template.OwnerUserID = trimString(template.OwnerUserID)
	template.Name = trimString(template.Name)
	template.Category = trimString(template.Category)
	template.TrafficClass = defaultString(template.TrafficClass, "transactional")
	template.Subject = trimString(template.Subject)
	template.FromEmail = trimString(template.FromEmail)
	template.ToEmail = trimString(template.ToEmail)
	template.Status = defaultString(template.Status, "draft")
	template.Variables = normalizeStringSlice(template.Variables)
	template.ConsumerID = trimString(template.ConsumerID)
	template.TextBody = strings.TrimSpace(template.TextBody)
	template.HTMLBody = strings.TrimSpace(template.HTMLBody)
	template.RetryMaxAttempts = maxInt(template.RetryMaxAttempts, 1)
	template.RetryBackoffSeconds = maxInt(template.RetryBackoffSeconds, 1)

	if template.WorkspaceID == "" {
		return smtp_errorx.ErrWorkspaceRequired
	}
	if template.Name == "" || template.Category == "" || template.Subject == "" || template.FromEmail == "" || template.ToEmail == "" || template.TextBody == "" {
		return smtp_errorx.ErrInvalidResource
	}

	if template.ConsumerID != "" {
		if s.consumerRepo == nil {
			return smtp_errorx.ErrInvalidResource
		}
		consumer, err := s.consumerRepo.GetConsumerByID(ctx, template.ConsumerID)
		if err != nil {
			return err
		}
		if consumer.WorkspaceID != template.WorkspaceID {
			return smtp_errorx.ErrWorkspaceMismatch
		}
	}

	return nil
}
