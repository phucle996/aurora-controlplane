package smtp_handler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"controlplane/internal/http/response"
	smtp_domainsvc "controlplane/internal/smtp/domain/service"
	smtp_resdto "controlplane/internal/smtp/transport/http/dto/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

type AggregationHandler struct {
	svc smtp_domainsvc.AggregationService
}

func NewAggregationHandler(svc smtp_domainsvc.AggregationService) *AggregationHandler {
	return &AggregationHandler{svc: svc}
}

var (
	throughputPointPool sync.Pool
	queueMixItemPool    sync.Pool
	timelineItemPool    sync.Pool
)

// @BasePath /api/v1/smtp/aggregation
// @Summary Get Workspace Aggregation
// @Description Get workspace aggregation
// @Tags smtp-aggregation
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/smtp/aggregation [get]
func (h *AggregationHandler) GetWorkspaceAggregation(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	// Workspace cookie scopes the read to one tenant; missing cookie is a bad request.
	workspaceID, cookieErr := c.Cookie("workspace_id")
	if cookieErr != nil {
		logger.HandlerError(c, "smtp.aggregation.get", cookieErr)
		response.RespondBadRequest(c, "workspace id is required")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		logger.HandlerError(c, "smtp.aggregation.get", errors.New("workspace id is required"))
		response.RespondBadRequest(c, "workspace id is required")
		return
	}

	// Aggregation is a read path; repo only returns context or wrapped DB failures here.
	item, err := h.svc.GetWorkspaceAggregation(ctx, workspaceID)
	if err != nil {
		logger.HandlerError(c, "smtp.aggregation.get", err)
		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			response.RespondServiceUnavailable(c, "smtp aggregation temporarily unavailable")
		default:
			response.RespondInternalError(c, "smtp aggregation failed")
		}
		return
	}

	logger.HandlerInfo(c, "smtp.aggregation.get", "smtp workspace aggregation fetched")
	borrowThroughputPoints := func(minCap int) []*smtp_resdto.OverviewThroughputPoint {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := throughputPointPool.Get().([]*smtp_resdto.OverviewThroughputPoint); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.OverviewThroughputPoint, 0, minCap)
	}
	releaseThroughputPoints := func(items []*smtp_resdto.OverviewThroughputPoint) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		throughputPointPool.Put(full[:0])
	}
	borrowQueueMixItems := func(minCap int) []*smtp_resdto.OverviewQueueMixItem {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := queueMixItemPool.Get().([]*smtp_resdto.OverviewQueueMixItem); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.OverviewQueueMixItem, 0, minCap)
	}
	releaseQueueMixItems := func(items []*smtp_resdto.OverviewQueueMixItem) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		queueMixItemPool.Put(full[:0])
	}
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
		full := items[:cap(items)]
		clear(full)
		gatewayListItemPool.Put(full[:0])
	}
	borrowTimelineItems := func(minCap int) []*smtp_resdto.OverviewTimelineItem {
		if minCap < pooledSliceDefaultCap {
			minCap = pooledSliceDefaultCap
		}
		if pooled, ok := timelineItemPool.Get().([]*smtp_resdto.OverviewTimelineItem); ok && cap(pooled) >= minCap {
			return pooled[:0]
		}
		return make([]*smtp_resdto.OverviewTimelineItem, 0, minCap)
	}
	releaseTimelineItems := func(items []*smtp_resdto.OverviewTimelineItem) {
		if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
			return
		}
		full := items[:cap(items)]
		clear(full)
		timelineItemPool.Put(full[:0])
	}
	throughput := borrowThroughputPoints(len(item.DeliveryThroughput))
	for _, point := range item.DeliveryThroughput {
		if point == nil {
			throughput = append(throughput, nil)
			continue
		}
		throughput = append(throughput, &smtp_resdto.OverviewThroughputPoint{
			Label:     point.Label,
			Delivered: point.Delivered,
			Queued:    point.Queued,
			Retries:   point.Retries,
		})
	}
	defer releaseThroughputPoints(throughput)

	queueMix := borrowQueueMixItems(len(item.QueueMix))
	for _, mix := range item.QueueMix {
		if mix == nil {
			queueMix = append(queueMix, nil)
			continue
		}
		queueMix = append(queueMix, &smtp_resdto.OverviewQueueMixItem{
			Category:   mix.Category,
			Pending:    mix.Pending,
			Processing: mix.Processing,
			Retries:    mix.Retries,
		})
	}
	defer releaseQueueMixItems(queueMix)

	gateways := borrowGatewayListItems(len(item.Gateways))
	for _, gateway := range item.Gateways {
		if gateway == nil {
			gateways = append(gateways, nil)
			continue
		}
		gateways = append(gateways, &smtp_resdto.GatewayListItem{
			ID:                  gateway.ID,
			Name:                gateway.Name,
			TrafficClass:        gateway.TrafficClass,
			Status:              gateway.Status,
			RoutingMode:         gateway.RoutingMode,
			Priority:            gateway.Priority,
			DesiredShardCount:   gateway.DesiredShardCount,
			TemplateCount:       gateway.TemplateCount,
			EndpointCount:       gateway.EndpointCount,
			ReadyShards:         gateway.ReadyShards,
			PendingShards:       gateway.PendingShards,
			DrainingShards:      gateway.DrainingShards,
			FallbackGatewayName: gateway.FallbackGatewayName,
			UpdatedAt:           gateway.UpdatedAt,
		})
	}
	defer releaseGatewayListItems(gateways)

	timeline := borrowTimelineItems(len(item.Timeline))
	for _, entry := range item.Timeline {
		if entry == nil {
			timeline = append(timeline, nil)
			continue
		}
		timeline = append(timeline, &smtp_resdto.OverviewTimelineItem{
			ID:         entry.ID,
			EntityType: entry.EntityType,
			EntityName: entry.EntityName,
			Action:     entry.Action,
			ActorName:  entry.ActorName,
			Note:       entry.Note,
			CreatedAt:  entry.CreatedAt,
		})
	}
	defer releaseTimelineItems(timeline)

	res := &smtp_resdto.SMTPOverview{
		Metrics: smtp_resdto.OverviewMetrics{
			DeliveredToday: item.Metrics.DeliveredToday,
			QueuedNow:      item.Metrics.QueuedNow,
			ActiveGateways: item.Metrics.ActiveGateways,
			TotalGateways:  item.Metrics.TotalGateways,
			LiveTemplates:  item.Metrics.LiveTemplates,
			TotalTemplates: item.Metrics.TotalTemplates,
		},
		DeliveryThroughput: throughput,
		HealthDistribution: smtp_resdto.OverviewHealthDistribution{
			Healthy: item.HealthDistribution.Healthy,
			Warning: item.HealthDistribution.Warning,
			Stopped: item.HealthDistribution.Stopped,
		},
		QueueMix: queueMix,
		Gateways: gateways,
		Timeline: timeline,
	}

	response.RespondSuccess(c, res, "ok")
}
