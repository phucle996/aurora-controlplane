package core_handler

import (
	"context"
	"errors"
	"time"

	"controlplane/internal/core/domain/entity"
	core_domainsvc "controlplane/internal/core/domain/service"
	core_errorx "controlplane/internal/core/errorx"
	core_reqdto "controlplane/internal/core/transport/http/dto/request"
	core_resdto "controlplane/internal/core/transport/http/dto/response"
	"controlplane/internal/http/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	svc core_domainsvc.WorkspaceService
}

func NewWorkspaceHandler(svc core_domainsvc.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

func (h *WorkspaceHandler) ListWorkspaceOptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	items, err := h.svc.ListWorkspaceOptions(ctx)
	if err != nil {
		logger.HandlerError(c, "core.workspace.options", err)
		response.RespondInternalError(c, "failed to list workspaces")
		return
	}

	logger.HandlerInfo(c, "core.workspace.options", "workspace options listed")
	response.RespondSuccess(c, gin.H{"items": items}, "ok")
}

func (h *WorkspaceHandler) ListWorkspaces(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.ListWorkspacesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.HandlerWarn(c, "core.workspace.list", err, "invalid request query")
		response.RespondBadRequest(c, "invalid request query")
		return
	}

	page, err := h.svc.ListWorkspaces(ctx, entity.WorkspaceListFilter{
		Page:     req.Page,
		Limit:    req.Limit,
		Query:    req.Query,
		Status:   req.Status,
		TenantID: req.TenantID,
	})
	if err != nil {
		logger.HandlerError(c, "core.workspace.list", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.workspace.list", "workspaces listed")
	response.RespondSuccess(c, core_resdto.WorkspacePageFromEntity(page), "ok")
}

func (h *WorkspaceHandler) GetWorkspace(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	item, err := h.svc.GetWorkspace(ctx, c.Param("id"))
	if err != nil {
		logger.HandlerError(c, "core.workspace.get", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.workspace.get", "workspace fetched")
	response.RespondSuccess(c, core_resdto.WorkspaceFromEntity(item), "ok")
}

func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "core.workspace.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	item, err := h.svc.CreateWorkspace(ctx, &entity.Workspace{
		Name:     req.Name,
		Status:   req.Status,
		TenantID: req.TenantID,
	})
	if err != nil {
		logger.HandlerError(c, "core.workspace.create", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.workspace.create", "workspace created")
	response.RespondCreated(c, core_resdto.WorkspaceFromEntity(item), "workspace created")
}

func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "core.workspace.update", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	item, err := h.svc.UpdateWorkspace(ctx, c.Param("id"), entity.WorkspacePatch{
		Name:   req.Name,
		Status: req.Status,
	})
	if err != nil {
		logger.HandlerError(c, "core.workspace.update", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.workspace.update", "workspace updated")
	response.RespondSuccess(c, core_resdto.WorkspaceFromEntity(item), "workspace updated")
}

func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.svc.DeleteWorkspace(ctx, c.Param("id")); err != nil {
		logger.HandlerError(c, "core.workspace.delete", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.workspace.delete", "workspace deleted")
	response.RespondSuccess(c, nil, "workspace deleted")
}

func (h *WorkspaceHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, core_errorx.ErrWorkspaceInvalid):
		response.RespondBadRequest(c, "invalid workspace request")
	case errors.Is(err, core_errorx.ErrWorkspaceNotFound):
		response.RespondNotFound(c, "workspace not found")
	case errors.Is(err, core_errorx.ErrWorkspaceAlreadyExists):
		response.RespondConflict(c, "workspace already exists")
	case errors.Is(err, core_errorx.ErrTenantNotFound):
		response.RespondNotFound(c, "tenant not found")
	default:
		response.RespondInternalError(c, "core workspace operation failed")
	}
}
