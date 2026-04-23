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

type TenantHandler struct {
	svc core_domainsvc.TenantService
}

func NewTenantHandler(svc core_domainsvc.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) ListTenants(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.ListTenantsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.HandlerWarn(c, "core.tenant.list", err, "invalid request query")
		response.RespondBadRequest(c, "invalid request query")
		return
	}

	page, err := h.svc.ListTenants(ctx, entity.TenantListFilter{
		Page:   req.Page,
		Limit:  req.Limit,
		Query:  req.Query,
		Status: req.Status,
	})
	if err != nil {
		logger.HandlerError(c, "core.tenant.list", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.tenant.list", "tenants listed")
	response.RespondSuccess(c, core_resdto.TenantPageFromEntity(page), "ok")
}

func (h *TenantHandler) GetTenant(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	item, err := h.svc.GetTenant(ctx, c.Param("id"))
	if err != nil {
		logger.HandlerError(c, "core.tenant.get", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.tenant.get", "tenant fetched")
	response.RespondSuccess(c, core_resdto.TenantFromEntity(item), "ok")
}

func (h *TenantHandler) CreateTenant(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "core.tenant.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	item, err := h.svc.CreateTenant(ctx, &entity.Tenant{
		Name:   req.Name,
		Status: req.Status,
	})
	if err != nil {
		logger.HandlerError(c, "core.tenant.create", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.tenant.create", "tenant created")
	response.RespondCreated(c, core_resdto.TenantFromEntity(item), "tenant created")
}

func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "core.tenant.update", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	item, err := h.svc.UpdateTenant(ctx, c.Param("id"), entity.TenantPatch{
		Name:   req.Name,
		Status: req.Status,
	})
	if err != nil {
		logger.HandlerError(c, "core.tenant.update", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.tenant.update", "tenant updated")
	response.RespondSuccess(c, core_resdto.TenantFromEntity(item), "tenant updated")
}

func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.svc.DeleteTenant(ctx, c.Param("id")); err != nil {
		logger.HandlerError(c, "core.tenant.delete", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.tenant.delete", "tenant deleted")
	response.RespondSuccess(c, nil, "tenant deleted")
}

func (h *TenantHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, core_errorx.ErrTenantInvalid):
		response.RespondBadRequest(c, "invalid tenant request")
	case errors.Is(err, core_errorx.ErrTenantNotFound):
		response.RespondNotFound(c, "tenant not found")
	case errors.Is(err, core_errorx.ErrTenantAlreadyExists):
		response.RespondConflict(c, "tenant already exists")
	default:
		response.RespondInternalError(c, "core tenant operation failed")
	}
}
