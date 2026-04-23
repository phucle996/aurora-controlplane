package smtp_handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"controlplane/internal/http/response"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_errorx "controlplane/internal/smtp/errorx"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type RuntimeHandler struct {
	svc smtp_domainsvc.RuntimeService
}

func NewRuntimeHandler(svc smtp_domainsvc.RuntimeService) *RuntimeHandler {
	return &RuntimeHandler{svc: svc}
}

// @BasePath /api/v1/smtp/runtime/activity-logs
// @Summary List Activity Logs
// @Description List activity logs
// @Tags smtp-runtime
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/runtime/activity-logs [get]
func (h *RuntimeHandler) ListActivityLogs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Runtime activity logs are tenant-scoped, so workspace cookie is mandatory.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.runtime.activity-logs", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.runtime.activity-logs", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListActivityLogs(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.runtime.activity-logs", err)
		// Map only runtime read errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.runtime.activity-logs", "smtp activity logs listed")
	response.RespondSuccess(c, items, "ok")
}

func (h *RuntimeHandler) ListDeliveryAttempts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Runtime delivery attempts are tenant-scoped, so workspace cookie is mandatory.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.runtime.delivery-attempts", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.runtime.delivery-attempts", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListDeliveryAttempts(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.runtime.delivery-attempts", err)
		// Map only runtime read errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.runtime.delivery-attempts", "smtp delivery attempts listed")
	response.RespondSuccess(c, items, "ok")
}

// @BasePath /api/v1/smtp/runtime/heartbeats
// @Summary List Heartbeats
// @Description List heartbeats
// @Tags smtp-runtime
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/runtime/heartbeats [get]
func (h *RuntimeHandler) ListRuntimeHeartbeats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	items, err := h.svc.ListRuntimeHeartbeats(ctx)
	if err != nil {
		logger.HandlerError(c, "smtp.runtime.heartbeats", err)
		// Heartbeat list is global runtime state; only runtime/container errors should surface here.
		switch {
		case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.runtime.heartbeats", "smtp runtime heartbeats listed")
	response.RespondSuccess(c, items, "ok")
}

// @BasePath /api/v1/smtp/runtime/gateway-assignments
// @Summary List Gateway Assignments
// @Description List gateway assignments
// @Tags smtp-runtime
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/runtime/gateway-assignments [get]
func (h *RuntimeHandler) ListGatewayAssignments(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	items, err := h.svc.ListGatewayAssignments(ctx)
	if err != nil {
		logger.HandlerError(c, "smtp.runtime.gateway-assignments", err)
		// Assignment list is global runtime state; only runtime/container errors should surface here.
		switch {
		case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.runtime.gateway-assignments", "smtp gateway assignments listed")
	response.RespondSuccess(c, items, "ok")
}

// @BasePath /api/v1/smtp/runtime/consumer-assignments
// @Summary List Consumer Assignments
// @Description List consumer assignments
// @Tags smtp-runtime
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/runtime/consumer-assignments [get]
func (h *RuntimeHandler) ListConsumerAssignments(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	items, err := h.svc.ListConsumerAssignments(ctx)
	if err != nil {
		logger.HandlerError(c, "smtp.runtime.consumer-assignments", err)
		// Assignment list is global runtime state; only runtime/container errors should surface here.
		switch {
		case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.runtime.consumer-assignments", "smtp consumer assignments listed")
	response.RespondSuccess(c, items, "ok")
}

// @BasePath /api/v1/smtp/runtime/reconcile
// @Summary Reconcile Runtime
// @Description Reconcile runtime
// @Tags smtp-runtime
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/runtime/reconcile [post]
func (h *RuntimeHandler) Reconcile(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := h.svc.Reconcile(ctx); err != nil {
		logger.HandlerError(c, "smtp.runtime.reconcile", err)
		// Reconcile only depends on runtime internals; map to runtime-specific response set.
		switch {
		case errors.Is(err, smtp_errorx.ErrRuntimeInvalid):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.runtime.reconcile", "smtp runtime reconciled")
	response.RespondSuccess(c, nil, "runtime reconciled")
}
