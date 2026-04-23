package core_handler

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"controlplane/internal/core/domain/entity"
	core_domainsvc "controlplane/internal/core/domain/service"
	core_errorx "controlplane/internal/core/errorx"
	"controlplane/internal/core/transport/http/dto/request"
	"controlplane/internal/http/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type ZoneHandler struct {
	svc core_domainsvc.ZoneService
}

func NewZoneHandler(svc core_domainsvc.ZoneService) *ZoneHandler {
	return &ZoneHandler{svc: svc}
}

func (h *ZoneHandler) ListZones(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	zones, err := h.svc.ListZones(ctx)
	if err != nil {
		logger.HandlerError(c, "core.zone.list", err)
		response.RespondInternalError(c, "failed to list zones")
		return
	}

	logger.HandlerInfo(c, "core.zone.list", "zones listed")
	response.RespondSuccess(c, zones, "ok")
}

func (h *ZoneHandler) GetZone(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	zone, err := h.svc.GetZone(ctx, c.Param("id"))
	if err != nil {
		logger.HandlerError(c, "core.zone.get", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.zone.get", "zone fetched")
	response.RespondSuccess(c, zone, "ok")
}

func (h *ZoneHandler) CreateZone(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.CreateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "core.zone.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		logger.HandlerWarn(c, "core.zone.create", nil, "invalid zone name")
		response.RespondBadRequest(c, "invalid zone name")
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" || utf8.RuneCountInString(slug) > 100 {
		logger.HandlerWarn(c, "core.zone.create", nil, "invalid zone slug")
		response.RespondBadRequest(c, "invalid zone slug")
		return
	}

	description := ""
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}

	zone := &entity.Zone{
		Slug:        slug,
		Name:        name,
		Description: description,
	}
	if err := h.svc.CreateZone(ctx, zone); err != nil {
		logger.HandlerError(c, "core.zone.create", err)
		response.RespondInternalError(c, "failed to create zone")
		return
	}

	logger.HandlerInfo(c, "core.zone.create", "zone created")
	response.RespondCreated(c, zone, "zone created")
}

func (h *ZoneHandler) UpdateZoneDescription(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req core_reqdto.UpdateZoneDescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "core.zone.update-description", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}
	if req.Description == nil {
		logger.HandlerWarn(c, "core.zone.update-description", nil, "missing zone description")
		response.RespondBadRequest(c, "description is required")
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(*req.Description)) > 2000 {
		logger.HandlerWarn(c, "core.zone.update-description", nil, "invalid zone description")
		response.RespondBadRequest(c, "invalid zone description")
		return
	}

	zone, err := h.svc.UpdateZoneDescription(ctx, c.Param("id"), *req.Description)
	if err != nil {
		logger.HandlerError(c, "core.zone.update-description", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.zone.update-description", "zone description updated")
	response.RespondSuccess(c, zone, "zone updated")
}

func (h *ZoneHandler) DeleteZone(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.svc.DeleteZone(ctx, c.Param("id")); err != nil {
		logger.HandlerError(c, "core.zone.delete", err)
		h.mapError(c, err)
		return
	}

	logger.HandlerInfo(c, "core.zone.delete", "zone deleted")
	response.RespondSuccess(c, nil, "zone deleted")
}

func (h *ZoneHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, core_errorx.ErrZoneNotFound):
		response.RespondNotFound(c, "zone not found")
	case errors.Is(err, core_errorx.ErrZoneInUse):
		response.RespondConflict(c, "zone is attached to dataplanes")
	default:
		response.RespondInternalError(c, "core zone operation failed")
	}
}
