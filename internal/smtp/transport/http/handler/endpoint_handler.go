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

type EndpointHandler struct {
	svc smtp_domainsvc.EndpointService
}

func NewEndpointHandler(svc smtp_domainsvc.EndpointService) *EndpointHandler {
	return &EndpointHandler{svc: svc}
}

// Pool keeps endpoint list allocations stable on high-frequency reads.
var endpointViewPool sync.Pool

// @BasePath /api/v1/smtp/endpoints
// @Summary List SMTP Endpoints
// @Description List SMTP endpoints
// @Tags smtp-endpoints
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/endpoints [get]
func (h *EndpointHandler) ListEndpoints(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.endpoint.list", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.endpoint.list", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListEndpointViews(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.endpoint.list", err)
		// Map only endpoint-flow errors produced by this service path.
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

	logger.HandlerInfo(c, "smtp.endpoint.list", "smtp endpoints listed")
	// Reuse response slice buffers to reduce allocation churn on list-heavy traffic.
	borrowEndpointViews := func(minCap int) []*smtp_resdto.EndpointView {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := endpointViewPool.Get().([]*smtp_resdto.EndpointView); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.EndpointView, 0, minCap)
	}
	releaseEndpointViews := func(items []*smtp_resdto.EndpointView) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		// Clear full backing array before pooling to prevent stale pointers.
		full := items[:cap(items)]
		clear(full)
		endpointViewPool.Put(full[:0])
	}
	views := borrowEndpointViews(len(items))
	for _, item := range items {
		if item == nil {
			views = append(views, nil)
			continue
		}
		views = append(views, &smtp_resdto.EndpointView{
			ID:                   item.ID,
			Name:                 item.Name,
			ProviderKind:         item.ProviderKind,
			Host:                 item.Host,
			Port:                 item.Port,
			Username:             item.Username,
			Priority:             item.Priority,
			Weight:               item.Weight,
			MaxConnections:       item.MaxConnections,
			MaxParallelSends:     item.MaxParallelSends,
			MaxMessagesPerSecond: item.MaxMessagesPerSecond,
			Burst:                item.Burst,
			WarmupState:          item.WarmupState,
			Status:               item.Status,
			TLSMode:              item.TLSMode,
			HasSecret:            item.HasSecret,
			HasCACert:            item.HasCACert,
			HasClientCert:        item.HasClientCert,
			HasClientKey:         item.HasClientKey,
			CreatedAt:            item.CreatedAt,
			UpdatedAt:            item.UpdatedAt,
		})
	}
	defer releaseEndpointViews(views)

	response.RespondSuccess(c, gin.H{"items": views}, "ok")
}

// @BasePath /api/v1/smtp/endpoints/:id
// @Summary Get SMTP Endpoint
// @Description Get SMTP endpoint
// @Tags smtp-endpoints
// @Accept json
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/endpoints/:id [get]
func (h *EndpointHandler) GetEndpoint(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil || workspaceID == "" {
		logger.HandlerError(c, "smtp.endpoint.get", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	endpointID := c.Param("id")
	if endpointID == "" {
		logger.HandlerWarn(c, "smtp.endpoint.get", nil, "invalid endpoint id")
		response.RespondBadRequest(c, "invalid endpoint id")
		return
	}

	item, err := h.svc.GetEndpointView(ctx, workspaceID, endpointID)
	if err != nil {
		logger.HandlerError(c, "smtp.endpoint.get", err)
		// Map only endpoint-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
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

	logger.HandlerInfo(c, "smtp.endpoint.get", "smtp endpoint fetched")
	response.RespondSuccess(c, &smtp_resdto.EndpointView{
		ID:                   item.ID,
		Name:                 item.Name,
		ProviderKind:         item.ProviderKind,
		Host:                 item.Host,
		Port:                 item.Port,
		Username:             item.Username,
		Priority:             item.Priority,
		Weight:               item.Weight,
		MaxConnections:       item.MaxConnections,
		MaxParallelSends:     item.MaxParallelSends,
		MaxMessagesPerSecond: item.MaxMessagesPerSecond,
		Burst:                item.Burst,
		WarmupState:          item.WarmupState,
		Status:               item.Status,
		TLSMode:              item.TLSMode,
		HasSecret:            item.HasSecret,
		HasCACert:            item.HasCACert,
		HasClientCert:        item.HasClientCert,
		HasClientKey:         item.HasClientKey,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}, "ok")
}

// @BasePath /api/v1/smtp/endpoints/try-connect
// @Summary Try Connect SMTP Endpoint
// @Description Try Connect SMTP endpoint
// @Tags smtp-endpoints
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/endpoints/try-connect [post]
func (h *EndpointHandler) TryConnect(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.endpoint.try_connect", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.endpoint.try_connect", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	var req smtp_reqdto.EndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.endpoint.try_connect", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	caCertPEM := ""
	if req.CACertPEM != nil {
		caCertPEM = *req.CACertPEM
	}
	clientCertPEM := ""
	if req.ClientCertPEM != nil {
		clientCertPEM = *req.ClientCertPEM
	}
	clientKeyPEM := ""
	if req.ClientKeyPEM != nil {
		clientKeyPEM = *req.ClientKeyPEM
	}
	secretRef := ""
	if req.SecretRef != nil {
		secretRef = *req.SecretRef
	}
	secretProvider := ""
	if req.SecretProvider != nil {
		secretProvider = *req.SecretProvider
	}

	item := &entity.Endpoint{
		WorkspaceID:          workspaceID,
		OwnerUserID:          userID,
		Name:                 req.Name,
		ProviderKind:         req.ProviderKind,
		Host:                 req.Host,
		Port:                 req.Port,
		Username:             req.Username,
		Priority:             req.Priority,
		Weight:               req.Weight,
		MaxConnections:       req.MaxConnections,
		MaxParallelSends:     req.MaxParallelSends,
		MaxMessagesPerSecond: req.MaxMessagesPerSecond,
		Burst:                req.Burst,
		WarmupState:          req.WarmupState,
		Status:               req.Status,
		TLSMode:              req.TLSMode,
		Password:             req.Password,
		CACertPEM:            caCertPEM,
		ClientCertPEM:        clientCertPEM,
		ClientKeyPEM:         clientKeyPEM,
		SecretRef:            secretRef,
		SecretProvider:       secretProvider,
	}

	if err := h.svc.TryConnect(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.endpoint.try_connect", err)
		// Map only endpoint-flow errors produced by this service path.
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

	logger.HandlerInfo(c, "smtp.endpoint.try_connect", "smtp endpoint connection succeeded")
	response.RespondSuccess(c, nil, "endpoint connection succeeded")
}

// @BasePath /api/v1/smtp/endpoints
// @Summary Create SMTP Endpoint
// @Description Create SMTP endpoint
// @Tags smtp-endpoints
// @Accept json
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/endpoints [post]
func (h *EndpointHandler) CreateEndpoint(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.endpoint.create", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.endpoint.create", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	var req smtp_reqdto.EndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.endpoint.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	caCertPEM := ""
	if req.CACertPEM != nil {
		caCertPEM = *req.CACertPEM
	}
	clientCertPEM := ""
	if req.ClientCertPEM != nil {
		clientCertPEM = *req.ClientCertPEM
	}
	clientKeyPEM := ""
	if req.ClientKeyPEM != nil {
		clientKeyPEM = *req.ClientKeyPEM
	}
	secretRef := ""
	if req.SecretRef != nil {
		secretRef = *req.SecretRef
	}
	secretProvider := ""
	if req.SecretProvider != nil {
		secretProvider = *req.SecretProvider
	}

	item := &entity.Endpoint{
		WorkspaceID:          workspaceID,
		OwnerUserID:          userID,
		Name:                 req.Name,
		ProviderKind:         req.ProviderKind,
		Host:                 req.Host,
		Port:                 req.Port,
		Username:             req.Username,
		Priority:             req.Priority,
		Weight:               req.Weight,
		MaxConnections:       req.MaxConnections,
		MaxParallelSends:     req.MaxParallelSends,
		MaxMessagesPerSecond: req.MaxMessagesPerSecond,
		Burst:                req.Burst,
		WarmupState:          req.WarmupState,
		Status:               req.Status,
		TLSMode:              req.TLSMode,
		Password:             req.Password,
		CACertPEM:            caCertPEM,
		ClientCertPEM:        clientCertPEM,
		ClientKeyPEM:         clientKeyPEM,
		SecretRef:            secretRef,
		SecretProvider:       secretProvider,
	}

	if err := h.svc.CreateEndpoint(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.endpoint.create", err)
		// Map only endpoint-flow errors produced by this service path.
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

	view, err := h.svc.GetEndpointView(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.endpoint.create", err)
		// Read-back keeps response shape consistent with GetEndpoint.
		switch {
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
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

	logger.HandlerInfo(c, "smtp.endpoint.create", "smtp endpoint created")
	response.RespondCreated(c, &smtp_resdto.EndpointView{
		ID:                   view.ID,
		Name:                 view.Name,
		ProviderKind:         view.ProviderKind,
		Host:                 view.Host,
		Port:                 view.Port,
		Username:             view.Username,
		Priority:             view.Priority,
		Weight:               view.Weight,
		MaxConnections:       view.MaxConnections,
		MaxParallelSends:     view.MaxParallelSends,
		MaxMessagesPerSecond: view.MaxMessagesPerSecond,
		Burst:                view.Burst,
		WarmupState:          view.WarmupState,
		Status:               view.Status,
		TLSMode:              view.TLSMode,
		HasSecret:            view.HasSecret,
		HasCACert:            view.HasCACert,
		HasClientCert:        view.HasClientCert,
		HasClientKey:         view.HasClientKey,
		CreatedAt:            view.CreatedAt,
		UpdatedAt:            view.UpdatedAt,
	}, "endpoint created")
}

// @BasePath /api/v1/smtp/endpoints/:id
// @Summary Update SMTP Endpoint
// @Description Update SMTP endpoint
// @Tags smtp-endpoints
// @Accept json
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/endpoints/:id [put]
func (h *EndpointHandler) UpdateEndpoint(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.endpoint.update", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.endpoint.update", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	var req smtp_reqdto.EndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.endpoint.update", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	caCertPEM := ""
	if req.CACertPEM != nil {
		caCertPEM = *req.CACertPEM
	}
	clientCertPEM := ""
	if req.ClientCertPEM != nil {
		clientCertPEM = *req.ClientCertPEM
	}
	clientKeyPEM := ""
	if req.ClientKeyPEM != nil {
		clientKeyPEM = *req.ClientKeyPEM
	}
	secretRef := ""
	if req.SecretRef != nil {
		secretRef = *req.SecretRef
	}
	secretProvider := ""
	if req.SecretProvider != nil {
		secretProvider = *req.SecretProvider
	}

	item := &entity.Endpoint{
		ID:                   c.Param("id"),
		WorkspaceID:          workspaceID,
		OwnerUserID:          userID,
		Name:                 req.Name,
		ProviderKind:         req.ProviderKind,
		Host:                 req.Host,
		Port:                 req.Port,
		Username:             req.Username,
		Priority:             req.Priority,
		Weight:               req.Weight,
		MaxConnections:       req.MaxConnections,
		MaxParallelSends:     req.MaxParallelSends,
		MaxMessagesPerSecond: req.MaxMessagesPerSecond,
		Burst:                req.Burst,
		WarmupState:          req.WarmupState,
		Status:               req.Status,
		TLSMode:              req.TLSMode,
		Password:             req.Password,
		CACertPEM:            caCertPEM,
		ClientCertPEM:        clientCertPEM,
		ClientKeyPEM:         clientKeyPEM,
		SecretRef:            secretRef,
		SecretProvider:       secretProvider,
	}

	if err := h.svc.UpdateEndpoint(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.endpoint.update", err)
		// Map only endpoint-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
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

	view, err := h.svc.GetEndpointView(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.endpoint.update", err)
		// Read-back keeps response shape consistent with GetEndpoint.
		switch {
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
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

	logger.HandlerInfo(c, "smtp.endpoint.update", "smtp endpoint updated")
	response.RespondSuccess(c, &smtp_resdto.EndpointView{
		ID:                   view.ID,
		Name:                 view.Name,
		ProviderKind:         view.ProviderKind,
		Host:                 view.Host,
		Port:                 view.Port,
		Username:             view.Username,
		Priority:             view.Priority,
		Weight:               view.Weight,
		MaxConnections:       view.MaxConnections,
		MaxParallelSends:     view.MaxParallelSends,
		MaxMessagesPerSecond: view.MaxMessagesPerSecond,
		Burst:                view.Burst,
		WarmupState:          view.WarmupState,
		Status:               view.Status,
		TLSMode:              view.TLSMode,
		HasSecret:            view.HasSecret,
		HasCACert:            view.HasCACert,
		HasClientCert:        view.HasClientCert,
		HasClientKey:         view.HasClientKey,
		CreatedAt:            view.CreatedAt,
		UpdatedAt:            view.UpdatedAt,
	}, "endpoint updated")
}

// @BasePath /api/v1/smtp/endpoints/:id
// @Summary Delete SMTP Endpoint
// @Description Delete SMTP endpoint
// @Tags smtp-endpoints
// @Accept json
// @Produce json
// @Param id path string true "Endpoint ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/endpoints/:id [delete]
func (h *EndpointHandler) DeleteEndpoint(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.endpoint.delete", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.endpoint.delete", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.endpoint.delete", smtp_errorx.ErrInvalidEndpointID)
		response.RespondBadRequest(c, "invalid endpoint id")
		return
	}

	if err := h.svc.DeleteEndpoint(ctx, workspaceID, itemID); err != nil {
		logger.HandlerError(c, "smtp.endpoint.delete", err)
		// Map only endpoint-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
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

	logger.HandlerInfo(c, "smtp.endpoint.delete", "smtp endpoint deleted")
	response.RespondSuccess(c, nil, "endpoint deleted")
}
