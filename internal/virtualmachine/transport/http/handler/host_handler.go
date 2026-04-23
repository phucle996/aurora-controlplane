package handler

import (
	"context"
	"errors"
	"time"

	"controlplane/internal/http/response"
	"controlplane/internal/virtualmachine/domain/entity"
	vm_domainsvc "controlplane/internal/virtualmachine/domain/service"
	vm_errorx "controlplane/internal/virtualmachine/errorx"
	vm_reqdto "controlplane/internal/virtualmachine/transport/http/dto/request"
	vm_resdto "controlplane/internal/virtualmachine/transport/http/dto/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type HostHandler struct {
	svc vm_domainsvc.HostService
}

func NewHostHandler(svc vm_domainsvc.HostService) *HostHandler {
	return &HostHandler{svc: svc}
}

func (h *HostHandler) ListHosts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req vm_reqdto.ListHostsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.HandlerWarn(c, "virtual-machine.host.list", err, "invalid request query")
		response.RespondBadRequest(c, "invalid request query")
		return
	}

	page, err := h.svc.ListHosts(ctx, entity.HostListFilter{
		Page:     req.Page,
		Limit:    req.Limit,
		Query:    req.Query,
		Status:   req.Status,
		ZoneSlug: req.ZoneSlug,
	})
	if err != nil {
		logger.HandlerError(c, "virtual-machine.host.list", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "virtual-machine.host.list", "hosts listed")
	response.RespondSuccess(c, vm_resdto.HostPageFromEntity(page), "ok")
}

func (h *HostHandler) GetHost(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	item, err := h.svc.GetHost(ctx, c.Param("host_id"))
	if err != nil {
		logger.HandlerError(c, "virtual-machine.host.get", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "virtual-machine.host.get", "host fetched")
	response.RespondSuccess(c, vm_resdto.HostFromEntity(item), "ok")
}

func (h *HostHandler) ListHostOptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req vm_reqdto.ListHostsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.HandlerWarn(c, "virtual-machine.host.options", err, "invalid request query")
		response.RespondBadRequest(c, "invalid request query")
		return
	}

	items, err := h.svc.ListHostOptions(ctx, entity.HostListFilter{
		Query:    req.Query,
		Status:   req.Status,
		ZoneSlug: req.ZoneSlug,
	})
	if err != nil {
		logger.HandlerError(c, "virtual-machine.host.options", err)
		h.mapError(c, err)
		return
	}

	options := make([]*vm_resdto.HostOption, 0, len(items))
	for _, item := range items {
		if mapped := vm_resdto.HostOptionFromEntity(item); mapped != nil {
			options = append(options, mapped)
		}
	}

	logger.HandlerInfo(c, "virtual-machine.host.options", "host options listed")
	response.RespondSuccess(c, gin.H{"items": options}, "ok")
}

func (h *HostHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, vm_errorx.ErrHostNotFound):
		response.RespondNotFound(c, "host not found")
	case errors.Is(err, vm_errorx.ErrHostInvalid):
		response.RespondBadRequest(c, "invalid host request")
	case errors.Is(err, vm_errorx.ErrHostConflict):
		response.RespondConflict(c, "host already bound to another agent")
	default:
		response.RespondInternalError(c, "virtual-machine host operation failed")
	}
}
