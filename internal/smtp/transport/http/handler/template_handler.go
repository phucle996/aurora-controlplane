package smtp_handler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"controlplane/internal/http/middleware"
	"controlplane/internal/http/response"
	"controlplane/internal/smtp/domain/entity"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
	smtp_reqdto "controlplane/internal/smtp/transport/http/dto/request"
	smtp_resdto "controlplane/internal/smtp/transport/http/dto/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type TemplateHandler struct {
	svc smtp_domainsvc.TemplateService
}

func NewTemplateHandler(svc smtp_domainsvc.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

var templateListItemPool sync.Pool

// @BasePath /api/v1/smtp/templates
// @Summary List SMTP Templates
// @Description List SMTP templates
// @Tags smtp-templates
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/templates [get]
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.template.list", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.template.list", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListTemplateItems(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.template.list", err)
		// Map only template-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.template.list", "smtp templates listed")
	borrowTemplateListItems := func(minCap int) []*smtp_resdto.TemplateListItem {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := templateListItemPool.Get().([]*smtp_resdto.TemplateListItem); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.TemplateListItem, 0, minCap)
	}
	releaseTemplateListItems := func(items []*smtp_resdto.TemplateListItem) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		templateListItemPool.Put(full[:0])
	}
	views := borrowTemplateListItems(len(items))
	for _, item := range items {
		if item == nil {
			views = append(views, nil)
			continue
		}
		views = append(views, &smtp_resdto.TemplateListItem{
			ID:           item.ID,
			Name:         item.Name,
			Category:     item.Category,
			TrafficClass: item.TrafficClass,
			Subject:      item.Subject,
			FromEmail:    item.FromEmail,
			ToEmail:      item.ToEmail,
			Status:       item.Status,
			ConsumerID:   item.ConsumerID,
			ConsumerName: item.ConsumerName,
			UpdatedAt:    item.UpdatedAt,
		})
	}
	defer releaseTemplateListItems(views)

	response.RespondSuccess(c, gin.H{"items": views}, "ok")
}

// @BasePath /api/v1/smtp/templates/:id
// @Summary Get SMTP Template
// @Description Get SMTP template
// @Tags smtp-templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/templates/:id [get]
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.template.get", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.template.get", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.template.get", smtp_errorx.ErrInvalidTemplateID)
		response.RespondBadRequest(c, "invalid template id")
		return
	}

	item, err := h.svc.GetTemplateDetail(ctx, workspaceID, itemID)
	if err != nil {
		logger.HandlerError(c, "smtp.template.get", err)
		// Map only template-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.template.get", "smtp template fetched")
	res := &smtp_resdto.TemplateDetail{
		ID:             item.ID,
		Name:           item.Name,
		Category:       item.Category,
		TrafficClass:   item.TrafficClass,
		Subject:        item.Subject,
		FromEmail:      item.FromEmail,
		ToEmail:        item.ToEmail,
		Status:         item.Status,
		Variables:      item.Variables,
		ConsumerID:     item.ConsumerID,
		ConsumerName:   item.ConsumerName,
		TextBody:       item.TextBody,
		HTMLBody:       item.HTMLBody,
		ActiveVersion:  item.ActiveVersion,
		RuntimeVersion: item.RuntimeVersion,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}

	response.RespondSuccess(c, res, "ok")
}

// @BasePath /api/v1/smtp/templates
// @Summary Create SMTP Template
// @Description Create SMTP template
// @Tags smtp-templates
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.template.create", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.template.create", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	ownerUserID := middleware.GetUserID(c)
	if ownerUserID == "" {
		logger.HandlerError(c, "smtp.template.create", smtp_errorx.ErrInvalidUserID)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	var req smtp_reqdto.TemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.template.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	item := &entity.Template{
		WorkspaceID:         workspaceID,
		OwnerUserID:         ownerUserID,
		Name:                req.Name,
		Category:            req.Category,
		TrafficClass:        req.TrafficClass,
		Subject:             req.Subject,
		FromEmail:           req.FromEmail,
		ToEmail:             req.ToEmail,
		Status:              req.Status,
		Variables:           req.Variables,
		ConsumerID:          req.ConsumerID,
		RetryMaxAttempts:    req.RetryMaxAttempts,
		RetryBackoffSeconds: req.RetryBackoffSeconds,
		TextBody:            req.TextBody,
		HTMLBody:            req.HTMLBody,
	}

	if err := h.svc.CreateTemplate(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.template.create", err)
		// Map template mutation errors from normalize + repository checks.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	detail, err := h.svc.GetTemplateDetail(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.template.create", err)
		// Read-after-write should normally succeed; map domain errors explicitly for diagnosis.
		switch {
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.template.create", "smtp template created")
	res := &smtp_resdto.TemplateDetail{
		ID:             detail.ID,
		Name:           detail.Name,
		Category:       detail.Category,
		TrafficClass:   detail.TrafficClass,
		Subject:        detail.Subject,
		FromEmail:      detail.FromEmail,
		ToEmail:        detail.ToEmail,
		Status:         detail.Status,
		Variables:      detail.Variables,
		ConsumerID:     detail.ConsumerID,
		ConsumerName:   detail.ConsumerName,
		TextBody:       detail.TextBody,
		HTMLBody:       detail.HTMLBody,
		ActiveVersion:  detail.ActiveVersion,
		RuntimeVersion: detail.RuntimeVersion,
		CreatedAt:      detail.CreatedAt,
		UpdatedAt:      detail.UpdatedAt,
	}

	response.RespondCreated(c, res, "template created")
}

// @BasePath /api/v1/smtp/templates/:id
// @Summary Update SMTP Template
// @Description Update SMTP template
// @Tags smtp-templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/templates/:id [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.template.update", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.template.update", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	var req smtp_reqdto.TemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.template.update", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	ownerUserID := middleware.GetUserID(c)
	if ownerUserID == "" {
		logger.HandlerError(c, "smtp.template.update", smtp_errorx.ErrInvalidUserID)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.template.update", smtp_errorx.ErrInvalidTemplateID)
		response.RespondBadRequest(c, "invalid template id")
		return
	}

	item := &entity.Template{
		ID:                  itemID,
		WorkspaceID:         workspaceID,
		OwnerUserID:         ownerUserID,
		Name:                req.Name,
		Category:            req.Category,
		TrafficClass:        req.TrafficClass,
		Subject:             req.Subject,
		FromEmail:           req.FromEmail,
		ToEmail:             req.ToEmail,
		Status:              req.Status,
		Variables:           req.Variables,
		ConsumerID:          req.ConsumerID,
		RetryMaxAttempts:    req.RetryMaxAttempts,
		RetryBackoffSeconds: req.RetryBackoffSeconds,
		TextBody:            req.TextBody,
		HTMLBody:            req.HTMLBody,
	}

	if err := h.svc.UpdateTemplate(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.template.update", err)
		// Map template mutation errors from normalize + repository checks.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	detail, err := h.svc.GetTemplateDetail(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.template.update", err)
		// Read-after-write should normally succeed; map domain errors explicitly for diagnosis.
		switch {
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.template.update", "smtp template updated")
	res := &smtp_resdto.TemplateDetail{
		ID:             detail.ID,
		Name:           detail.Name,
		Category:       detail.Category,
		TrafficClass:   detail.TrafficClass,
		Subject:        detail.Subject,
		FromEmail:      detail.FromEmail,
		ToEmail:        detail.ToEmail,
		Status:         detail.Status,
		Variables:      detail.Variables,
		ConsumerID:     detail.ConsumerID,
		ConsumerName:   detail.ConsumerName,
		TextBody:       detail.TextBody,
		HTMLBody:       detail.HTMLBody,
		ActiveVersion:  detail.ActiveVersion,
		RuntimeVersion: detail.RuntimeVersion,
		CreatedAt:      detail.CreatedAt,
		UpdatedAt:      detail.UpdatedAt,
	}

	response.RespondSuccess(c, res, "template updated")
}

// @BasePath /api/v1/smtp/templates/:id
// @Summary Delete SMTP Template
// @Description Delete SMTP template
// @Tags smtp-templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/templates/:id [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.template.delete", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.template.delete", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.template.delete", smtp_errorx.ErrInvalidTemplateID)
		response.RespondBadRequest(c, "invalid template id")
		return
	}

	if err := h.svc.DeleteTemplate(ctx, workspaceID, itemID); err != nil {
		logger.HandlerError(c, "smtp.template.delete", err)
		// Map only template delete errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.template.delete", "smtp template deleted")
	response.RespondSuccess(c, nil, "template deleted")
}
