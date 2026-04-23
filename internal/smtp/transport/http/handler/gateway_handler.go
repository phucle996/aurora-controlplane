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

type GatewayHandler struct {
	svc smtp_domainsvc.GatewayService
}

func NewGatewayHandler(svc smtp_domainsvc.GatewayService) *GatewayHandler {
	return &GatewayHandler{svc: svc}
}

const (
	pooledSliceDefaultCap = 16
	pooledSliceMaxCap     = 512
)

var (
	// Pools keep gateway read responses stable under list/detail hot traffic.
	gatewayListItemPool sync.Pool
	gatewayTemplatePool sync.Pool
	gatewayEndpointPool sync.Pool
)

// @BasePath /api/v1/smtp/gateways
// @Summary List SMTP Gateways
// @Description List SMTP gateways
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways [get]
func (h *GatewayHandler) ListGateways(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.list", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.list", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListGatewayItems(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.gateway.list", err)
		// Map only gateway-flow errors produced by this service path.
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

	logger.HandlerInfo(c, "smtp.gateway.list", "smtp gateways listed")
	// Reuse response slice buffers to reduce allocation churn on list-heavy traffic.
	borrowGatewayListItems := func(minCap int) []*smtp_resdto.GatewayListItem {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayListItemPool.Get().([]*smtp_resdto.GatewayListItem); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayListItem, 0, minCap)
	}
	releaseGatewayListItems := func(items []*smtp_resdto.GatewayListItem) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		// Clear full backing array before pooling to prevent stale pointers.
		full := items[:cap(items)]
		clear(full)
		gatewayListItemPool.Put(full[:0])
	}
	views := borrowGatewayListItems(len(items))
	for _, item := range items {
		if item == nil {
			views = append(views, nil)
			continue
		}
		views = append(views, &smtp_resdto.GatewayListItem{
			ID:                  item.ID,
			Name:                item.Name,
			TrafficClass:        item.TrafficClass,
			Status:              item.Status,
			RoutingMode:         item.RoutingMode,
			Priority:            item.Priority,
			DesiredShardCount:   item.DesiredShardCount,
			TemplateCount:       item.TemplateCount,
			EndpointCount:       item.EndpointCount,
			ReadyShards:         item.ReadyShards,
			PendingShards:       item.PendingShards,
			DrainingShards:      item.DrainingShards,
			FallbackGatewayName: item.FallbackGatewayName,
			UpdatedAt:           item.UpdatedAt,
		})
	}
	defer releaseGatewayListItems(views)

	response.RespondSuccess(c, gin.H{"items": views}, "ok")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Get SMTP Gateway
// @Description Get SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id [get]
func (h *GatewayHandler) GetGateway(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.get", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.get", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.gateway.get", smtp_errorx.ErrInvalidGatewayID)
		response.RespondBadRequest(c, "invalid gateway id")
		return
	}

	item, err := h.svc.GetGatewayDetail(ctx, workspaceID, itemID)
	if err != nil {
		logger.HandlerError(c, "smtp.gateway.get", err)
		// Map only gateway-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
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

	logger.HandlerInfo(c, "smtp.gateway.get", "smtp gateway fetched")
	borrowGatewayTemplateBindings := func(minCap int) []*smtp_resdto.GatewayTemplateBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayTemplatePool.Get().([]*smtp_resdto.GatewayTemplateBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayTemplateBinding, 0, minCap)
	}
	releaseGatewayTemplateBindings := func(items []*smtp_resdto.GatewayTemplateBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayTemplatePool.Put(full[:0])
	}
	borrowGatewayEndpointBindings := func(minCap int) []*smtp_resdto.GatewayEndpointBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayEndpointPool.Get().([]*smtp_resdto.GatewayEndpointBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayEndpointBinding, 0, minCap)
	}
	releaseGatewayEndpointBindings := func(items []*smtp_resdto.GatewayEndpointBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayEndpointPool.Put(full[:0])
	}

	templates := borrowGatewayTemplateBindings(len(item.Templates))
	for _, binding := range item.Templates {
		if binding == nil {
			templates = append(templates, nil)
			continue
		}
		templates = append(templates, &smtp_resdto.GatewayTemplateBinding{
			ID:           binding.ID,
			Name:         binding.Name,
			Category:     binding.Category,
			TrafficClass: binding.TrafficClass,
			Status:       binding.Status,
			ConsumerID:   binding.ConsumerID,
			ConsumerName: binding.ConsumerName,
			Selected:     binding.Selected,
			Position:     binding.Position,
		})
	}
	defer releaseGatewayTemplateBindings(templates)

	endpoints := borrowGatewayEndpointBindings(len(item.Endpoints))
	for _, binding := range item.Endpoints {
		if binding == nil {
			endpoints = append(endpoints, nil)
			continue
		}
		endpoints = append(endpoints, &smtp_resdto.GatewayEndpointBinding{
			ID:       binding.ID,
			Name:     binding.Name,
			Host:     binding.Host,
			Port:     binding.Port,
			Username: binding.Username,
			Status:   binding.Status,
			Selected: binding.Selected,
			Position: binding.Position,
		})
	}
	defer releaseGatewayEndpointBindings(endpoints)

	res := &smtp_resdto.GatewayDetail{
		ID:                item.ID,
		Name:              item.Name,
		TrafficClass:      item.TrafficClass,
		Status:            item.Status,
		RoutingMode:       item.RoutingMode,
		Priority:          item.Priority,
		DesiredShardCount: item.DesiredShardCount,
		RuntimeVersion:    item.RuntimeVersion,
		FallbackGateway:   nil,
		Templates:         templates,
		Endpoints:         endpoints,
		ReadyShards:       item.ReadyShards,
		PendingShards:     item.PendingShards,
		DrainingShards:    item.DrainingShards,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
	if item.FallbackGateway != nil {
		res.FallbackGateway = &smtp_resdto.GatewayFallbackSummary{
			ID:     item.FallbackGateway.ID,
			Name:   item.FallbackGateway.Name,
			Status: item.FallbackGateway.Status,
		}
	}

	response.RespondSuccess(c, res, "ok")
}

func (h *GatewayHandler) GetGatewayDetail(c *gin.Context) {
	h.GetGateway(c)
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Update SMTP Gateway Templates
// @Description Update SMTP gateway templates
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id/templates [put]
func (h *GatewayHandler) UpdateGatewayTemplates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var req smtp_reqdto.GatewayTemplateBindingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.gateway.templates", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.templates", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.templates", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.gateway.templates", smtp_errorx.ErrInvalidGatewayID)
		response.RespondBadRequest(c, "invalid gateway id")
		return
	}

	item, err := h.svc.UpdateGatewayTemplates(ctx, workspaceID, itemID, req.TemplateIDs)
	if err != nil {
		logger.HandlerError(c, "smtp.gateway.templates", err)
		// Map only gateway mutation errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, smtp_errorx.ErrZoneMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same zone")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.gateway.templates", "smtp gateway templates updated")
	borrowGatewayTemplateBindings := func(minCap int) []*smtp_resdto.GatewayTemplateBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayTemplatePool.Get().([]*smtp_resdto.GatewayTemplateBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayTemplateBinding, 0, minCap)
	}
	releaseGatewayTemplateBindings := func(items []*smtp_resdto.GatewayTemplateBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayTemplatePool.Put(full[:0])
	}
	borrowGatewayEndpointBindings := func(minCap int) []*smtp_resdto.GatewayEndpointBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayEndpointPool.Get().([]*smtp_resdto.GatewayEndpointBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayEndpointBinding, 0, minCap)
	}
	releaseGatewayEndpointBindings := func(items []*smtp_resdto.GatewayEndpointBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayEndpointPool.Put(full[:0])
	}
	templates := borrowGatewayTemplateBindings(len(item.Templates))
	for _, binding := range item.Templates {
		if binding == nil {
			templates = append(templates, nil)
			continue
		}
		templates = append(templates, &smtp_resdto.GatewayTemplateBinding{
			ID:           binding.ID,
			Name:         binding.Name,
			Category:     binding.Category,
			TrafficClass: binding.TrafficClass,
			Status:       binding.Status,
			ConsumerID:   binding.ConsumerID,
			ConsumerName: binding.ConsumerName,
			Selected:     binding.Selected,
			Position:     binding.Position,
		})
	}
	defer releaseGatewayTemplateBindings(templates)

	endpoints := borrowGatewayEndpointBindings(len(item.Endpoints))
	for _, binding := range item.Endpoints {
		if binding == nil {
			endpoints = append(endpoints, nil)
			continue
		}
		endpoints = append(endpoints, &smtp_resdto.GatewayEndpointBinding{
			ID:       binding.ID,
			Name:     binding.Name,
			Host:     binding.Host,
			Port:     binding.Port,
			Username: binding.Username,
			Status:   binding.Status,
			Selected: binding.Selected,
			Position: binding.Position,
		})
	}
	defer releaseGatewayEndpointBindings(endpoints)

	res := &smtp_resdto.GatewayDetail{
		ID:                item.ID,
		Name:              item.Name,
		TrafficClass:      item.TrafficClass,
		Status:            item.Status,
		RoutingMode:       item.RoutingMode,
		Priority:          item.Priority,
		DesiredShardCount: item.DesiredShardCount,
		RuntimeVersion:    item.RuntimeVersion,
		FallbackGateway:   nil,
		Templates:         templates,
		Endpoints:         endpoints,
		ReadyShards:       item.ReadyShards,
		PendingShards:     item.PendingShards,
		DrainingShards:    item.DrainingShards,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
	if item.FallbackGateway != nil {
		res.FallbackGateway = &smtp_resdto.GatewayFallbackSummary{
			ID:     item.FallbackGateway.ID,
			Name:   item.FallbackGateway.Name,
			Status: item.FallbackGateway.Status,
		}
	}

	response.RespondSuccess(c, res, "gateway templates updated")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Update SMTP Gateway Endpoints
// @Description Update SMTP gateway endpoints
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id/endpoints [put]
func (h *GatewayHandler) UpdateGatewayEndpoints(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var req smtp_reqdto.GatewayEndpointBindingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.gateway.endpoints", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.endpoints", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.endpoints", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.gateway.endpoints", smtp_errorx.ErrInvalidGatewayID)
		response.RespondBadRequest(c, "invalid gateway id")
		return
	}

	item, err := h.svc.UpdateGatewayEndpoints(ctx, workspaceID, itemID, req.EndpointIDs)
	if err != nil {
		logger.HandlerError(c, "smtp.gateway.endpoints", err)
		// Map only gateway mutation errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, smtp_errorx.ErrZoneMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same zone")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.gateway.endpoints", "smtp gateway endpoints updated")
	borrowGatewayTemplateBindings := func(minCap int) []*smtp_resdto.GatewayTemplateBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayTemplatePool.Get().([]*smtp_resdto.GatewayTemplateBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayTemplateBinding, 0, minCap)
	}
	releaseGatewayTemplateBindings := func(items []*smtp_resdto.GatewayTemplateBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayTemplatePool.Put(full[:0])
	}
	borrowGatewayEndpointBindings := func(minCap int) []*smtp_resdto.GatewayEndpointBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayEndpointPool.Get().([]*smtp_resdto.GatewayEndpointBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayEndpointBinding, 0, minCap)
	}
	releaseGatewayEndpointBindings := func(items []*smtp_resdto.GatewayEndpointBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayEndpointPool.Put(full[:0])
	}
	templates := borrowGatewayTemplateBindings(len(item.Templates))
	for _, binding := range item.Templates {
		if binding == nil {
			templates = append(templates, nil)
			continue
		}
		templates = append(templates, &smtp_resdto.GatewayTemplateBinding{
			ID:           binding.ID,
			Name:         binding.Name,
			Category:     binding.Category,
			TrafficClass: binding.TrafficClass,
			Status:       binding.Status,
			ConsumerID:   binding.ConsumerID,
			ConsumerName: binding.ConsumerName,
			Selected:     binding.Selected,
			Position:     binding.Position,
		})
	}
	defer releaseGatewayTemplateBindings(templates)

	endpoints := borrowGatewayEndpointBindings(len(item.Endpoints))
	for _, binding := range item.Endpoints {
		if binding == nil {
			endpoints = append(endpoints, nil)
			continue
		}
		endpoints = append(endpoints, &smtp_resdto.GatewayEndpointBinding{
			ID:       binding.ID,
			Name:     binding.Name,
			Host:     binding.Host,
			Port:     binding.Port,
			Username: binding.Username,
			Status:   binding.Status,
			Selected: binding.Selected,
			Position: binding.Position,
		})
	}
	defer releaseGatewayEndpointBindings(endpoints)

	res := &smtp_resdto.GatewayDetail{
		ID:                item.ID,
		Name:              item.Name,
		TrafficClass:      item.TrafficClass,
		Status:            item.Status,
		RoutingMode:       item.RoutingMode,
		Priority:          item.Priority,
		DesiredShardCount: item.DesiredShardCount,
		RuntimeVersion:    item.RuntimeVersion,
		FallbackGateway:   nil,
		Templates:         templates,
		Endpoints:         endpoints,
		ReadyShards:       item.ReadyShards,
		PendingShards:     item.PendingShards,
		DrainingShards:    item.DrainingShards,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
	if item.FallbackGateway != nil {
		res.FallbackGateway = &smtp_resdto.GatewayFallbackSummary{
			ID:     item.FallbackGateway.ID,
			Name:   item.FallbackGateway.Name,
			Status: item.FallbackGateway.Status,
		}
	}

	response.RespondSuccess(c, res, "gateway endpoints updated")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Start SMTP Gateway
// @Description Start SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id/start [post]
func (h *GatewayHandler) StartGateway(c *gin.Context) {
	h.handleGatewayStateMutation(c, "start")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Drain SMTP Gateway
// @Description Drain SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id/drain [post]
func (h *GatewayHandler) DrainGateway(c *gin.Context) {
	h.handleGatewayStateMutation(c, "drain")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Disable SMTP Gateway
// @Description Disable SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id/disable [post]
func (h *GatewayHandler) DisableGateway(c *gin.Context) {
	h.handleGatewayStateMutation(c, "disable")
}

// @BasePath /api/v1/smtp/gateways
// @Summary Create SMTP Gateway
// @Description Create SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways [post]
func (h *GatewayHandler) CreateGateway(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var req smtp_reqdto.GatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.gateway.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.create", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.create", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	ownerUserID := middleware.GetUserID(c)
	if ownerUserID == "" {
		logger.HandlerError(c, "smtp.gateway.create", smtp_errorx.ErrUnauthorized)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	item := &entity.Gateway{
		WorkspaceID:       workspaceID,
		OwnerUserID:       ownerUserID,
		ZoneID:            req.ZoneID,
		Name:              req.Name,
		TrafficClass:      req.TrafficClass,
		Status:            req.Status,
		RoutingMode:       req.RoutingMode,
		Priority:          req.Priority,
		FallbackGatewayID: req.FallbackGatewayID,
		DesiredShardCount: req.DesiredShardCount,
		TemplateIDs:       req.TemplateIDs,
		EndpointIDs:       req.EndpointIDs,
	}

	if err := h.svc.CreateGateway(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.gateway.create", err)
		// Map only gateway mutation errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, smtp_errorx.ErrZoneMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same zone")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	detail, err := h.svc.GetGatewayDetail(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.gateway.create", err)
		// Read-back keeps response shape consistent with GetGateway.
		switch {
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
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

	logger.HandlerInfo(c, "smtp.gateway.create", "smtp gateway created")
	borrowGatewayTemplateBindings := func(minCap int) []*smtp_resdto.GatewayTemplateBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayTemplatePool.Get().([]*smtp_resdto.GatewayTemplateBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayTemplateBinding, 0, minCap)
	}
	releaseGatewayTemplateBindings := func(items []*smtp_resdto.GatewayTemplateBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayTemplatePool.Put(full[:0])
	}
	borrowGatewayEndpointBindings := func(minCap int) []*smtp_resdto.GatewayEndpointBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayEndpointPool.Get().([]*smtp_resdto.GatewayEndpointBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayEndpointBinding, 0, minCap)
	}
	releaseGatewayEndpointBindings := func(items []*smtp_resdto.GatewayEndpointBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayEndpointPool.Put(full[:0])
	}
	templates := borrowGatewayTemplateBindings(len(detail.Templates))
	for _, binding := range detail.Templates {
		if binding == nil {
			templates = append(templates, nil)
			continue
		}
		templates = append(templates, &smtp_resdto.GatewayTemplateBinding{
			ID:           binding.ID,
			Name:         binding.Name,
			Category:     binding.Category,
			TrafficClass: binding.TrafficClass,
			Status:       binding.Status,
			ConsumerID:   binding.ConsumerID,
			ConsumerName: binding.ConsumerName,
			Selected:     binding.Selected,
			Position:     binding.Position,
		})
	}
	defer releaseGatewayTemplateBindings(templates)

	endpoints := borrowGatewayEndpointBindings(len(detail.Endpoints))
	for _, binding := range detail.Endpoints {
		if binding == nil {
			endpoints = append(endpoints, nil)
			continue
		}
		endpoints = append(endpoints, &smtp_resdto.GatewayEndpointBinding{
			ID:       binding.ID,
			Name:     binding.Name,
			Host:     binding.Host,
			Port:     binding.Port,
			Username: binding.Username,
			Status:   binding.Status,
			Selected: binding.Selected,
			Position: binding.Position,
		})
	}
	defer releaseGatewayEndpointBindings(endpoints)

	res := &smtp_resdto.GatewayDetail{
		ID:                detail.ID,
		Name:              detail.Name,
		TrafficClass:      detail.TrafficClass,
		Status:            detail.Status,
		RoutingMode:       detail.RoutingMode,
		Priority:          detail.Priority,
		DesiredShardCount: detail.DesiredShardCount,
		RuntimeVersion:    detail.RuntimeVersion,
		FallbackGateway:   nil,
		Templates:         templates,
		Endpoints:         endpoints,
		ReadyShards:       detail.ReadyShards,
		PendingShards:     detail.PendingShards,
		DrainingShards:    detail.DrainingShards,
		CreatedAt:         detail.CreatedAt,
		UpdatedAt:         detail.UpdatedAt,
	}
	if detail.FallbackGateway != nil {
		res.FallbackGateway = &smtp_resdto.GatewayFallbackSummary{
			ID:     detail.FallbackGateway.ID,
			Name:   detail.FallbackGateway.Name,
			Status: detail.FallbackGateway.Status,
		}
	}

	response.RespondCreated(c, res, "gateway created")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Update SMTP Gateway
// @Description Update SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id [put]
func (h *GatewayHandler) UpdateGateway(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var req smtp_reqdto.GatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.gateway.update", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.update", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.update", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.gateway.update", smtp_errorx.ErrInvalidGatewayID)
		response.RespondBadRequest(c, "invalid gateway id")
		return
	}

	ownerUserID := middleware.GetUserID(c)
	if ownerUserID == "" {
		logger.HandlerError(c, "smtp.gateway.update", smtp_errorx.ErrUnauthorized)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	item := &entity.Gateway{
		ID:                itemID,
		WorkspaceID:       workspaceID,
		OwnerUserID:       ownerUserID,
		ZoneID:            req.ZoneID,
		Name:              req.Name,
		TrafficClass:      req.TrafficClass,
		Status:            req.Status,
		RoutingMode:       req.RoutingMode,
		Priority:          req.Priority,
		FallbackGatewayID: req.FallbackGatewayID,
		DesiredShardCount: req.DesiredShardCount,
		TemplateIDs:       req.TemplateIDs,
		EndpointIDs:       req.EndpointIDs,
	}

	if err := h.svc.UpdateGateway(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.gateway.update", err)
		// Map only gateway mutation errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, smtp_errorx.ErrZoneMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same zone")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	detail, err := h.svc.GetGatewayDetail(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.gateway.update", err)
		// Read-back keeps response shape consistent with GetGateway.
		switch {
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
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

	logger.HandlerInfo(c, "smtp.gateway.update", "smtp gateway updated")
	borrowGatewayTemplateBindings := func(minCap int) []*smtp_resdto.GatewayTemplateBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayTemplatePool.Get().([]*smtp_resdto.GatewayTemplateBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayTemplateBinding, 0, minCap)
	}
	releaseGatewayTemplateBindings := func(items []*smtp_resdto.GatewayTemplateBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayTemplatePool.Put(full[:0])
	}
	borrowGatewayEndpointBindings := func(minCap int) []*smtp_resdto.GatewayEndpointBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayEndpointPool.Get().([]*smtp_resdto.GatewayEndpointBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayEndpointBinding, 0, minCap)
	}
	releaseGatewayEndpointBindings := func(items []*smtp_resdto.GatewayEndpointBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayEndpointPool.Put(full[:0])
	}
	templates := borrowGatewayTemplateBindings(len(detail.Templates))
	for _, binding := range detail.Templates {
		if binding == nil {
			templates = append(templates, nil)
			continue
		}
		templates = append(templates, &smtp_resdto.GatewayTemplateBinding{
			ID:           binding.ID,
			Name:         binding.Name,
			Category:     binding.Category,
			TrafficClass: binding.TrafficClass,
			Status:       binding.Status,
			ConsumerID:   binding.ConsumerID,
			ConsumerName: binding.ConsumerName,
			Selected:     binding.Selected,
			Position:     binding.Position,
		})
	}
	defer releaseGatewayTemplateBindings(templates)

	endpoints := borrowGatewayEndpointBindings(len(detail.Endpoints))
	for _, binding := range detail.Endpoints {
		if binding == nil {
			endpoints = append(endpoints, nil)
			continue
		}
		endpoints = append(endpoints, &smtp_resdto.GatewayEndpointBinding{
			ID:       binding.ID,
			Name:     binding.Name,
			Host:     binding.Host,
			Port:     binding.Port,
			Username: binding.Username,
			Status:   binding.Status,
			Selected: binding.Selected,
			Position: binding.Position,
		})
	}
	defer releaseGatewayEndpointBindings(endpoints)

	res := &smtp_resdto.GatewayDetail{
		ID:                detail.ID,
		Name:              detail.Name,
		TrafficClass:      detail.TrafficClass,
		Status:            detail.Status,
		RoutingMode:       detail.RoutingMode,
		Priority:          detail.Priority,
		DesiredShardCount: detail.DesiredShardCount,
		RuntimeVersion:    detail.RuntimeVersion,
		FallbackGateway:   nil,
		Templates:         templates,
		Endpoints:         endpoints,
		ReadyShards:       detail.ReadyShards,
		PendingShards:     detail.PendingShards,
		DrainingShards:    detail.DrainingShards,
		CreatedAt:         detail.CreatedAt,
		UpdatedAt:         detail.UpdatedAt,
	}
	if detail.FallbackGateway != nil {
		res.FallbackGateway = &smtp_resdto.GatewayFallbackSummary{
			ID:     detail.FallbackGateway.ID,
			Name:   detail.FallbackGateway.Name,
			Status: detail.FallbackGateway.Status,
		}
	}

	response.RespondSuccess(c, res, "gateway updated")
}

// @BasePath /api/v1/smtp/gateways/:id
// @Summary Delete SMTP Gateway
// @Description Delete SMTP gateway
// @Tags smtp-gateways
// @Accept json
// @Produce json
// @Param id path string true "Gateway ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/gateways/:id [delete]
func (h *GatewayHandler) DeleteGateway(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway.delete", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway.delete", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.gateway.delete", smtp_errorx.ErrInvalidGatewayID)
		response.RespondBadRequest(c, "invalid gateway id")
		return
	}

	if err := h.svc.DeleteGateway(ctx, workspaceID, itemID); err != nil {
		logger.HandlerError(c, "smtp.gateway.delete", err)
		// Map only gateway-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
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

	logger.HandlerInfo(c, "smtp.gateway.delete", "smtp gateway deleted")
	response.RespondSuccess(c, nil, "gateway deleted")
}

func (h *GatewayHandler) handleGatewayStateMutation(c *gin.Context, action string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.gateway."+action, cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.gateway."+action, smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	itemID := c.Param("id")
	if itemID == "" {
		logger.HandlerError(c, "smtp.gateway."+action, smtp_errorx.ErrInvalidGatewayID)
		response.RespondBadRequest(c, "invalid gateway id")
		return
	}

	var item *entity.GatewayDetail
	var err error
	switch action {
	case "start":
		item, err = h.svc.StartGateway(ctx, workspaceID, itemID)
	case "drain":
		item, err = h.svc.DrainGateway(ctx, workspaceID, itemID)
	case "disable":
		item, err = h.svc.DisableGateway(ctx, workspaceID, itemID)
	default:
		response.RespondBadRequest(c, "invalid request")
		return
	}
	if err != nil {
		logger.HandlerError(c, "smtp.gateway."+action, err)
		// Map only gateway mutation errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrTemplateNotFound):
			response.RespondNotFound(c, "template not found")
		case errors.Is(err, smtp_errorx.ErrGatewayNotFound):
			response.RespondNotFound(c, "gateway not found")
		case errors.Is(err, smtp_errorx.ErrEndpointNotFound):
			response.RespondNotFound(c, "endpoint not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, smtp_errorx.ErrWorkspaceMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same workspace")
		case errors.Is(err, smtp_errorx.ErrZoneMismatch):
			response.RespondConflict(c, "smtp resources must belong to the same zone")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.gateway."+action, "smtp gateway state updated")
	message := "gateway updated"
	switch action {
	case "start":
		message = "gateway started"
	case "drain":
		message = "gateway draining"
	case "disable":
		message = "gateway disabled"
	}
	borrowGatewayTemplateBindings := func(minCap int) []*smtp_resdto.GatewayTemplateBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayTemplatePool.Get().([]*smtp_resdto.GatewayTemplateBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayTemplateBinding, 0, minCap)
	}
	releaseGatewayTemplateBindings := func(items []*smtp_resdto.GatewayTemplateBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayTemplatePool.Put(full[:0])
	}
	borrowGatewayEndpointBindings := func(minCap int) []*smtp_resdto.GatewayEndpointBinding {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := gatewayEndpointPool.Get().([]*smtp_resdto.GatewayEndpointBinding); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.GatewayEndpointBinding, 0, minCap)
	}
	releaseGatewayEndpointBindings := func(items []*smtp_resdto.GatewayEndpointBinding) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		gatewayEndpointPool.Put(full[:0])
	}
	templates := borrowGatewayTemplateBindings(len(item.Templates))
	for _, binding := range item.Templates {
		if binding == nil {
			templates = append(templates, nil)
			continue
		}
		templates = append(templates, &smtp_resdto.GatewayTemplateBinding{
			ID:           binding.ID,
			Name:         binding.Name,
			Category:     binding.Category,
			TrafficClass: binding.TrafficClass,
			Status:       binding.Status,
			ConsumerID:   binding.ConsumerID,
			ConsumerName: binding.ConsumerName,
			Selected:     binding.Selected,
			Position:     binding.Position,
		})
	}
	defer releaseGatewayTemplateBindings(templates)

	endpoints := borrowGatewayEndpointBindings(len(item.Endpoints))
	for _, binding := range item.Endpoints {
		if binding == nil {
			endpoints = append(endpoints, nil)
			continue
		}
		endpoints = append(endpoints, &smtp_resdto.GatewayEndpointBinding{
			ID:       binding.ID,
			Name:     binding.Name,
			Host:     binding.Host,
			Port:     binding.Port,
			Username: binding.Username,
			Status:   binding.Status,
			Selected: binding.Selected,
			Position: binding.Position,
		})
	}
	defer releaseGatewayEndpointBindings(endpoints)

	res := &smtp_resdto.GatewayDetail{
		ID:                item.ID,
		Name:              item.Name,
		TrafficClass:      item.TrafficClass,
		Status:            item.Status,
		RoutingMode:       item.RoutingMode,
		Priority:          item.Priority,
		DesiredShardCount: item.DesiredShardCount,
		RuntimeVersion:    item.RuntimeVersion,
		FallbackGateway:   nil,
		Templates:         templates,
		Endpoints:         endpoints,
		ReadyShards:       item.ReadyShards,
		PendingShards:     item.PendingShards,
		DrainingShards:    item.DrainingShards,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
	if item.FallbackGateway != nil {
		res.FallbackGateway = &smtp_resdto.GatewayFallbackSummary{
			ID:     item.FallbackGateway.ID,
			Name:   item.FallbackGateway.Name,
			Status: item.FallbackGateway.Status,
		}
	}

	response.RespondSuccess(c, res, message)
}
