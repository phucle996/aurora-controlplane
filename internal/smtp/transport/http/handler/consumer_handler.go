package smtp_handler

import (
	"context"
	"encoding/json"
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

type ConsumerHandler struct {
	svc smtp_domainsvc.ConsumerService
}

func NewConsumerHandler(svc smtp_domainsvc.ConsumerService) *ConsumerHandler {
	return &ConsumerHandler{svc: svc}
}

var (
	// Pools keep hot list endpoints from reallocating large response slices on every request.
	consumerViewPool   sync.Pool
	consumerOptionPool sync.Pool
)

// @BasePath /api/v1/workspaces/:workspace_id/smtp
// @Summary List SMTP Consumers
// @Description List all SMTP consumers for a workspace
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers [get]
func (h *ConsumerHandler) ListConsumers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.consumer.list", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.list", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListConsumerViews(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.consumer.list", err)
		// Map only consumer-flow errors produced by this service path.
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

	logger.HandlerInfo(c, "smtp.consumer.list", "smtp consumers listed")
	// Reuse response slice buffers to reduce allocation churn on list-heavy traffic.
	borrowConsumerViews := func(minCap int) []*smtp_resdto.ConsumerView {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := consumerViewPool.Get().([]*smtp_resdto.ConsumerView); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.ConsumerView, 0, minCap)
	}
	releaseConsumerViews := func(items []*smtp_resdto.ConsumerView) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		// Clear full backing array before pooling to prevent stale pointers.
		full := items[:cap(items)]
		clear(full)
		consumerViewPool.Put(full[:0])
	}
	views := borrowConsumerViews(len(items))
	for _, item := range items {
		if item == nil {
			views = append(views, nil)
			continue
		}
		var connectionConfig json.RawMessage
		if len(item.ConnectionConfig) > 0 {
			connectionConfig = make([]byte, len(item.ConnectionConfig))
			copy(connectionConfig, item.ConnectionConfig)
		}
		views = append(views, &smtp_resdto.ConsumerView{
			ID:                item.ID,
			ZoneID:            item.ZoneID,
			Name:              item.Name,
			TransportType:     item.TransportType,
			Source:            item.Source,
			ConsumerGroup:     item.ConsumerGroup,
			WorkerConcurrency: item.WorkerConcurrency,
			AckTimeoutSeconds: item.AckTimeoutSeconds,
			BatchSize:         item.BatchSize,
			Status:            item.Status,
			Note:              item.Note,
			ConnectionConfig:  connectionConfig,
			DesiredShardCount: item.DesiredShardCount,
			HasSecret:         item.HasSecret,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
	}
	defer releaseConsumerViews(views)

	response.RespondSuccess(c, gin.H{"items": views}, "ok")
}

// @BasePath /api/v1/workspaces/:workspace_id/smtp
// @Summary Get SMTP Consumer
// @Description Get SMTP consumer by ID
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Param id path string true "Consumer ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers/{id} [get]
func (h *ConsumerHandler) GetConsumer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.consumer.get", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.get", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	consumerID := c.Param("id")
	if consumerID == "" {
		logger.HandlerError(c, "smtp.consumer.get", smtp_errorx.ErrInvalidConsumerID)
		response.RespondBadRequest(c, "invalid consumer id")
		return
	}

	item, err := h.svc.GetConsumerView(ctx, workspaceID, consumerID)
	if err != nil {
		logger.HandlerError(c, "smtp.consumer.get", err)
		// Map only consumer-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
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

	logger.HandlerInfo(c, "smtp.consumer.get", "smtp consumer fetched")
	var connectionConfig json.RawMessage
	if len(item.ConnectionConfig) > 0 {
		connectionConfig = make([]byte, len(item.ConnectionConfig))
		copy(connectionConfig, item.ConnectionConfig)
	}

	response.RespondSuccess(c, &smtp_resdto.ConsumerView{
		ID:                item.ID,
		ZoneID:            item.ZoneID,
		Name:              item.Name,
		TransportType:     item.TransportType,
		Source:            item.Source,
		ConsumerGroup:     item.ConsumerGroup,
		WorkerConcurrency: item.WorkerConcurrency,
		AckTimeoutSeconds: item.AckTimeoutSeconds,
		BatchSize:         item.BatchSize,
		Status:            item.Status,
		Note:              item.Note,
		ConnectionConfig:  connectionConfig,
		DesiredShardCount: item.DesiredShardCount,
		HasSecret:         item.HasSecret,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}, "ok")
}

// @BasePath /api/v1/smtp/consumers/options
// @Summary List Consumer Options
// @Description List consumer options
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers/options [get]
func (h *ConsumerHandler) ListConsumerOptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.consumer.options", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.options", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	items, err := h.svc.ListConsumerOptions(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.consumer.options", err)
		// Map only consumer-flow errors produced by this service path.
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

	logger.HandlerInfo(c, "smtp.consumer.options", "smtp consumer options listed")
	// Reuse response slice buffers to reduce allocation churn on list-heavy traffic.
	borrowConsumerOptions := func(minCap int) []*smtp_resdto.ConsumerOption {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := consumerOptionPool.Get().([]*smtp_resdto.ConsumerOption); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.ConsumerOption, 0, minCap)
	}
	releaseConsumerOptions := func(items []*smtp_resdto.ConsumerOption) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		// Clear full backing array before pooling to prevent stale pointers.
		full := items[:cap(items)]
		clear(full)
		consumerOptionPool.Put(full[:0])
	}
	options := borrowConsumerOptions(len(items))
	for _, item := range items {
		if item == nil {
			options = append(options, nil)
			continue
		}
		options = append(options, &smtp_resdto.ConsumerOption{
			ID:     item.ID,
			Label:  item.Label,
			Status: item.Status,
		})
	}
	defer releaseConsumerOptions(options)

	response.RespondSuccess(c, gin.H{"items": options}, "ok")
}

// @BasePath /api/v1/smtp/consumers/try-connect
// @Summary Try Connect SMTP Consumer
// @Description Try connect SMTP consumer
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers/try-connect [post]
func (h *ConsumerHandler) TryConnect(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil || workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.try_connect", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	var req smtp_reqdto.ConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.consumer.try_connect", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	OwnerUserID := middleware.GetUserID(c)
	if OwnerUserID == "" {
		logger.HandlerError(c, "smtp.consumer.try_connect", smtp_errorx.ErrInvalidUserID)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	item := &entity.Consumer{
		WorkspaceID:       workspaceID,
		OwnerUserID:       OwnerUserID,
		ZoneID:            req.ZoneID,
		Name:              req.Name,
		TransportType:     req.TransportType,
		Source:            req.Source,
		ConsumerGroup:     req.ConsumerGroup,
		WorkerConcurrency: req.WorkerConcurrency,
		AckTimeoutSeconds: req.AckTimeoutSeconds,
		BatchSize:         req.BatchSize,
		Status:            req.Status,
		Note:              req.Note,
		ConnectionConfig:  req.ConnectionConfig,
		DesiredShardCount: req.DesiredShardCount,
		SecretConfig:      req.SecretConfig,
		SecretRef:         req.SecretRef,
		SecretProvider:    req.SecretProvider,
	}

	if err := h.svc.TryConnect(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.consumer.try_connect", err)
		// Map only consumer-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.consumer.try_connect", "smtp consumer connection succeeded")
	response.RespondSuccess(c, nil, "consumer connection succeeded")
}

// @BasePath /api/v1/workspaces/:workspace_id/smtp
// @Summary Create SMTP Consumer
// @Description Create SMTP consumer
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Param req body smtp_reqdto.ConsumerRequest true "Consumer Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers [post]
func (h *ConsumerHandler) CreateConsumer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.consumer.create", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.create", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	var req smtp_reqdto.ConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.consumer.create", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	OwnerUserID := middleware.GetUserID(c)
	if OwnerUserID == "" {
		logger.HandlerError(c, "smtp.consumer.create", smtp_errorx.ErrInvalidUserID)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	item := &entity.Consumer{
		WorkspaceID:       workspaceID,
		OwnerUserID:       OwnerUserID,
		ZoneID:            req.ZoneID,
		Name:              req.Name,
		TransportType:     req.TransportType,
		Source:            req.Source,
		ConsumerGroup:     req.ConsumerGroup,
		WorkerConcurrency: req.WorkerConcurrency,
		AckTimeoutSeconds: req.AckTimeoutSeconds,
		BatchSize:         req.BatchSize,
		Status:            req.Status,
		Note:              req.Note,
		ConnectionConfig:  req.ConnectionConfig,
		DesiredShardCount: req.DesiredShardCount,
		SecretConfig:      req.SecretConfig,
		SecretRef:         req.SecretRef,
		SecretProvider:    req.SecretProvider,
	}

	if err := h.svc.CreateConsumer(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.consumer.create", err)
		// Map only consumer-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	view, err := h.svc.GetConsumerView(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.consumer.create", err)
		// Read-back keeps response shape consistent with GetConsumer.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
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

	logger.HandlerInfo(c, "smtp.consumer.create", "smtp consumer created")
	var connectionConfig json.RawMessage
	if len(view.ConnectionConfig) > 0 {
		connectionConfig = make([]byte, len(view.ConnectionConfig))
		copy(connectionConfig, view.ConnectionConfig)
	}
	res := &smtp_resdto.ConsumerView{
		ID:                view.ID,
		ZoneID:            view.ZoneID,
		Name:              view.Name,
		TransportType:     view.TransportType,
		Source:            view.Source,
		ConsumerGroup:     view.ConsumerGroup,
		WorkerConcurrency: view.WorkerConcurrency,
		AckTimeoutSeconds: view.AckTimeoutSeconds,
		BatchSize:         view.BatchSize,
		Status:            view.Status,
		Note:              view.Note,
		ConnectionConfig:  connectionConfig,
		DesiredShardCount: view.DesiredShardCount,
		HasSecret:         view.HasSecret,
		CreatedAt:         view.CreatedAt,
		UpdatedAt:         view.UpdatedAt,
	}

	response.RespondCreated(c, res, "consumer created")
}

// @BasePath /api/v1/smtp/consumers/:id
// @Summary Update SMTP Consumer
// @Description Update SMTP consumer
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Param req body smtp_reqdto.UpdateConsumerRequest true "Consumer Request"
// @Param id path string true "Consumer ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers/{id} [put]
func (h *ConsumerHandler) UpdateConsumer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.consumer.update", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.update", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	consumerID := c.Param("id")
	if consumerID == "" {
		logger.HandlerError(c, "smtp.consumer.update", smtp_errorx.ErrInvalidConsumerID)
		response.RespondBadRequest(c, "invalid consumer id")
		return
	}

	var req smtp_reqdto.ConsumerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.HandlerWarn(c, "smtp.consumer.update", err, "invalid request payload")
		response.RespondBadRequest(c, "invalid request payload")
		return
	}

	OwnerUserID := middleware.GetUserID(c)
	if OwnerUserID == "" {
		logger.HandlerError(c, "smtp.consumer.update", smtp_errorx.ErrInvalidUserID)
		response.RespondUnauthorized(c, "unauthorized")
		return
	}

	item := &entity.Consumer{
		ID:                consumerID,
		WorkspaceID:       workspaceID,
		OwnerUserID:       OwnerUserID,
		ZoneID:            req.ZoneID,
		Name:              req.Name,
		TransportType:     req.TransportType,
		Source:            req.Source,
		ConsumerGroup:     req.ConsumerGroup,
		WorkerConcurrency: req.WorkerConcurrency,
		AckTimeoutSeconds: req.AckTimeoutSeconds,
		BatchSize:         req.BatchSize,
		Status:            req.Status,
		Note:              req.Note,
		ConnectionConfig:  req.ConnectionConfig,
		DesiredShardCount: req.DesiredShardCount,
		SecretConfig:      req.SecretConfig,
		SecretRef:         req.SecretRef,
		SecretProvider:    req.SecretProvider,
	}

	if err := h.svc.UpdateConsumer(ctx, item); err != nil {
		logger.HandlerError(c, "smtp.consumer.update", err)
		// Map only consumer-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
		case errors.Is(err, smtp_errorx.ErrWorkspaceRequired):
			response.RespondBadRequest(c, "workspace is required")
		case errors.Is(err, smtp_errorx.ErrZoneRequired):
			response.RespondBadRequest(c, "zone is required")
		case errors.Is(err, smtp_errorx.ErrInvalidResource):
			response.RespondBadRequest(c, "invalid request")
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp operation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp operation failed")
		}
		return
	}

	view, err := h.svc.GetConsumerView(ctx, item.WorkspaceID, item.ID)
	if err != nil {
		logger.HandlerError(c, "smtp.consumer.update", err)
		// Read-back keeps response shape consistent with GetConsumer.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
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

	logger.HandlerInfo(c, "smtp.consumer.update", "smtp consumer updated")
	var connectionConfig json.RawMessage
	if len(view.ConnectionConfig) > 0 {
		connectionConfig = make([]byte, len(view.ConnectionConfig))
		copy(connectionConfig, view.ConnectionConfig)
	}
	res := &smtp_resdto.ConsumerView{
		ID:                view.ID,
		ZoneID:            view.ZoneID,
		Name:              view.Name,
		TransportType:     view.TransportType,
		Source:            view.Source,
		ConsumerGroup:     view.ConsumerGroup,
		WorkerConcurrency: view.WorkerConcurrency,
		AckTimeoutSeconds: view.AckTimeoutSeconds,
		BatchSize:         view.BatchSize,
		Status:            view.Status,
		Note:              view.Note,
		ConnectionConfig:  connectionConfig,
		DesiredShardCount: view.DesiredShardCount,
		HasSecret:         view.HasSecret,
		CreatedAt:         view.CreatedAt,
		UpdatedAt:         view.UpdatedAt,
	}

	response.RespondSuccess(c, res, "consumer updated")
}

// @BasePath /api/v1/smtp/consumers/:id
// @Summary Delete SMTP Consumer
// @Description Delete SMTP consumer
// @Tags smtp-consumers
// @Accept json
// @Produce json
// @Param id path string true "Consumer ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/consumers/{id} [delete]
func (h *ConsumerHandler) DeleteConsumer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace scope is a transport precondition and must be resolved before service calls.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.consumer.delete", cookieErr)
		response.RespondBadRequest(c, "workspace is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.consumer.delete", smtp_errorx.ErrWorkspaceRequired)
		response.RespondBadRequest(c, "workspace is required")
		return
	}

	consumerID := c.Param("id")
	if consumerID == "" {
		logger.HandlerError(c, "smtp.consumer.delete", smtp_errorx.ErrInvalidConsumerID)
		response.RespondBadRequest(c, "invalid consumer id")
		return
	}

	if err := h.svc.DeleteConsumer(ctx, workspaceID, consumerID); err != nil {
		logger.HandlerError(c, "smtp.consumer.delete", err)
		// Map only consumer-flow errors produced by this service path.
		switch {
		case errors.Is(err, smtp_errorx.ErrConsumerNotFound):
			response.RespondNotFound(c, "consumer not found")
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

	logger.HandlerInfo(c, "smtp.consumer.delete", "smtp consumer deleted")
	response.RespondSuccess(c, nil, "consumer deleted")
}
