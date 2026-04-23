package smtp_handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"controlplane/internal/smtp/domain/entity"
	smtp_resdto "controlplane/internal/smtp/transport/http/dto/response"
	"controlplane/pkg/logger"

	"github.com/gin-gonic/gin"
)

func TestPooledSliceHelpersClearAndReuse(t *testing.T) {
	tests := []struct {
		name    string
		borrow  func(int) []*smtp_resdto.TemplateListItem
		release func([]*smtp_resdto.TemplateListItem)
		sample  *smtp_resdto.TemplateListItem
	}{
		{
			name: "template list items",
			borrow: func(minCap int) []*smtp_resdto.TemplateListItem {
				if minCap < pooledSliceDefaultCap {
					minCap = pooledSliceDefaultCap
				}
				if pooled, ok := templateListItemPool.Get().([]*smtp_resdto.TemplateListItem); ok && cap(pooled) >= minCap {
					return pooled[:0]
				}
				return make([]*smtp_resdto.TemplateListItem, 0, minCap)
			},
			release: func(items []*smtp_resdto.TemplateListItem) {
				if cap(items) == 0 || cap(items) > pooledSliceMaxCap {
					return
				}
				full := items[:cap(items)]
				clear(full)
				templateListItemPool.Put(full[:0])
			},
			sample: &smtp_resdto.TemplateListItem{ID: "template-a", Name: "template-a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertPooledSliceRoundTrip(t, tc.borrow, tc.release, tc.sample)
		})
	}
}

func TestReleaseSliceDropsOversizeBackingArrays(t *testing.T) {
	var pool sync.Pool
	oversized := make([]*smtp_resdto.TemplateListItem, 1, pooledSliceMaxCap+1)

	if cap(oversized) == 0 || cap(oversized) > pooledSliceMaxCap {
		// drop the slice
	} else {
		full := oversized[:cap(oversized)]
		clear(full)
		pool.Put(full[:0])
	}

	if got := pool.Get(); got != nil {
		t.Fatalf("expected oversize slice to be dropped, got %T", got)
	}
}

func TestTemplateHandlerListTemplatesReturnsStableJSONAcrossCalls(t *testing.T) {
	initSMTPHandlerTests()

	svc := &stubTemplatePoolService{
		listItems: []*entity.TemplateListItem{
			{ID: "template-alpha", Name: "template-alpha", UpdatedAt: fixedTime},
		},
	}
	h := NewTemplateHandler(svc)

	first := invokeTemplateList(t, h)
	if !strings.Contains(first, "template-alpha") {
		t.Fatalf("expected first response to contain alpha template, got %s", first)
	}

	svc.listItems = []*entity.TemplateListItem{
		{ID: "template-beta", Name: "template-beta", UpdatedAt: fixedTime},
	}
	second := invokeTemplateList(t, h)
	if !strings.Contains(second, "template-beta") {
		t.Fatalf("expected second response to contain beta template, got %s", second)
	}
	if strings.Contains(second, "template-alpha") {
		t.Fatalf("expected second response to be free of stale alpha data, got %s", second)
	}
}

func TestGatewayHandlerListGatewaysReturnsStableJSONAcrossCalls(t *testing.T) {
	initSMTPHandlerTests()

	svc := &stubGatewayPoolService{
		listItems: []*entity.GatewayListItem{
			{ID: "gateway-alpha", Name: "gateway-alpha", UpdatedAt: fixedTime},
		},
	}
	h := NewGatewayHandler(svc)

	first := invokeGatewayList(t, h)
	if !strings.Contains(first, "gateway-alpha") {
		t.Fatalf("expected first response to contain alpha gateway, got %s", first)
	}

	svc.listItems = []*entity.GatewayListItem{
		{ID: "gateway-beta", Name: "gateway-beta", UpdatedAt: fixedTime},
	}
	second := invokeGatewayList(t, h)
	if !strings.Contains(second, "gateway-beta") {
		t.Fatalf("expected second response to contain beta gateway, got %s", second)
	}
	if strings.Contains(second, "gateway-alpha") {
		t.Fatalf("expected second response to be free of stale alpha data, got %s", second)
	}
}

func TestGatewayHandlerGetGatewayReturnsStableJSONAcrossCalls(t *testing.T) {
	initSMTPHandlerTests()

	svc := &stubGatewayPoolService{
		detail: sampleGatewayDetail("gateway-alpha"),
	}
	h := NewGatewayHandler(svc)

	first := invokeGatewayDetail(t, h)
	if !strings.Contains(first, "gateway-alpha") {
		t.Fatalf("expected first response to contain alpha gateway, got %s", first)
	}

	svc.detail = sampleGatewayDetail("gateway-beta")
	second := invokeGatewayDetail(t, h)
	if !strings.Contains(second, "gateway-beta") {
		t.Fatalf("expected second response to contain beta gateway, got %s", second)
	}
	if strings.Contains(second, "gateway-alpha") {
		t.Fatalf("expected second response to be free of stale alpha data, got %s", second)
	}
}

func TestAggregationHandlerGetWorkspaceAggregationReturnsStableJSONAcrossCalls(t *testing.T) {
	initSMTPHandlerTests()

	svc := &stubAggregationPoolService{
		item: sampleSMTPOverview("alpha"),
	}
	h := NewAggregationHandler(svc)

	first := invokeAggregation(t, h)
	if !strings.Contains(first, "gateway-alpha") {
		t.Fatalf("expected first response to contain alpha gateway, got %s", first)
	}

	svc.item = sampleSMTPOverview("beta")
	second := invokeAggregation(t, h)
	if !strings.Contains(second, "gateway-beta") {
		t.Fatalf("expected second response to contain beta gateway, got %s", second)
	}
	if strings.Contains(second, "gateway-alpha") {
		t.Fatalf("expected second response to be free of stale alpha data, got %s", second)
	}
}

func BenchmarkTemplateHandlerListTemplates(b *testing.B) {
	initSMTPHandlerTests()

	svc := &stubTemplatePoolService{
		listItems: sampleTemplateListItems("benchmark", 64),
	}
	h := NewTemplateHandler(svc)

	invokeTemplateList(b, h)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		invokeTemplateList(b, h)
	}
}

func BenchmarkGatewayHandlerListGateways(b *testing.B) {
	initSMTPHandlerTests()

	svc := &stubGatewayPoolService{
		listItems: sampleGatewayListItems("benchmark", 64),
	}
	h := NewGatewayHandler(svc)

	invokeGatewayList(b, h)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		invokeGatewayList(b, h)
	}
}

func BenchmarkGatewayHandlerGetGateway(b *testing.B) {
	initSMTPHandlerTests()

	svc := &stubGatewayPoolService{
		detail: sampleGatewayDetail("gateway-benchmark"),
	}
	h := NewGatewayHandler(svc)

	invokeGatewayDetail(b, h)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		invokeGatewayDetail(b, h)
	}
}

func BenchmarkAggregationHandlerGetWorkspaceAggregation(b *testing.B) {
	initSMTPHandlerTests()

	svc := &stubAggregationPoolService{
		item: sampleSMTPOverview("benchmark"),
	}
	h := NewAggregationHandler(svc)

	invokeAggregation(b, h)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		invokeAggregation(b, h)
	}
}

func assertPooledSliceRoundTrip[T any](
	t *testing.T,
	borrow func(int) []T,
	release func([]T),
	sample T,
) {
	t.Helper()

	first := borrow(1)
	first = append(first, sample)
	release(first)

	second := borrow(1)
	defer release(second)

	full := second[:cap(second)]
	var zero T
	for i, got := range full {
		if !reflect.DeepEqual(got, zero) {
			t.Fatalf("expected pooled backing slot %d to be cleared, got %#v", i, got)
		}
	}
}

func initSMTPHandlerTests() {
	gin.SetMode(gin.TestMode)
	logger.InitLogger()
}

func invokeTemplateList(tb testing.TB, h *TemplateHandler) string {
	tb.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/smtp/templates", nil)
	req.AddCookie(&http.Cookie{Name: "workspace_id", Value: "workspace-1"})
	c.Request = req

	h.ListTemplates(c)

	if w.Code != http.StatusOK {
		tb.Fatalf("expected 200 from template list, got %d (%s)", w.Code, w.Body.String())
	}

	return w.Body.String()
}

func invokeGatewayList(tb testing.TB, h *GatewayHandler) string {
	tb.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/smtp/gateways", nil)
	req.AddCookie(&http.Cookie{Name: "workspace_id", Value: "workspace-1"})
	c.Request = req

	h.ListGateways(c)

	if w.Code != http.StatusOK {
		tb.Fatalf("expected 200 from gateway list, got %d (%s)", w.Code, w.Body.String())
	}

	return w.Body.String()
}

func invokeGatewayDetail(tb testing.TB, h *GatewayHandler) string {
	tb.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/smtp/gateways/gateway-1", nil)
	req.AddCookie(&http.Cookie{Name: "workspace_id", Value: "workspace-1"})
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "gateway-1"}}

	h.GetGateway(c)

	if w.Code != http.StatusOK {
		tb.Fatalf("expected 200 from gateway detail, got %d (%s)", w.Code, w.Body.String())
	}

	return w.Body.String()
}

func invokeAggregation(tb testing.TB, h *AggregationHandler) string {
	tb.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/smtp/aggregation", nil)
	req.AddCookie(&http.Cookie{Name: "workspace_id", Value: "workspace-1"})
	c.Request = req

	h.GetWorkspaceAggregation(c)

	if w.Code != http.StatusOK {
		tb.Fatalf("expected 200 from aggregation, got %d (%s)", w.Code, w.Body.String())
	}

	return w.Body.String()
}

func sampleTemplateListItems(prefix string, count int) []*entity.TemplateListItem {
	items := make([]*entity.TemplateListItem, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, &entity.TemplateListItem{
			ID:           prefix + "-template-" + string(rune('a'+(i%26))),
			Name:         prefix + "-template-" + string(rune('a'+(i%26))),
			Category:     "marketing",
			TrafficClass: "default",
			Subject:      "subject",
			FromEmail:    "from@example.test",
			ToEmail:      "to@example.test",
			Status:       "active",
			ConsumerID:   "consumer-1",
			ConsumerName: "consumer-one",
			UpdatedAt:    fixedTime,
		})
	}
	return items
}

func sampleGatewayListItems(prefix string, count int) []*entity.GatewayListItem {
	items := make([]*entity.GatewayListItem, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, &entity.GatewayListItem{
			ID:                  prefix + "-gateway-" + string(rune('a'+(i%26))),
			Name:                prefix + "-gateway-" + string(rune('a'+(i%26))),
			TrafficClass:        "default",
			Status:              "active",
			RoutingMode:         "direct",
			Priority:            1,
			DesiredShardCount:   1,
			TemplateCount:       1,
			EndpointCount:       1,
			ReadyShards:         1,
			PendingShards:       0,
			DrainingShards:      0,
			FallbackGatewayName: "fallback",
			UpdatedAt:           fixedTime,
		})
	}
	return items
}

func sampleGatewayDetail(prefix string) *entity.GatewayDetail {
	return &entity.GatewayDetail{
		ID:                prefix,
		Name:              prefix,
		TrafficClass:      "default",
		Status:            "active",
		RoutingMode:       "direct",
		Priority:          10,
		DesiredShardCount: 2,
		RuntimeVersion:    1,
		FallbackGateway: &entity.GatewayFallbackSummary{
			ID:     prefix + "-fallback",
			Name:   prefix + "-fallback",
			Status: "active",
		},
		Templates: []*entity.GatewayTemplateBinding{
			{
				ID:           prefix + "-template",
				Name:         prefix + "-template",
				Category:     "marketing",
				TrafficClass: "default",
				Status:       "active",
				ConsumerID:   "consumer-1",
				ConsumerName: "consumer-one",
				Selected:     true,
				Position:     1,
			},
		},
		Endpoints: []*entity.GatewayEndpointBinding{
			{
				ID:       prefix + "-endpoint",
				Name:     prefix + "-endpoint",
				Host:     "smtp.example.test",
				Port:     2525,
				Username: "user",
				Status:   "active",
				Selected: true,
				Position: 1,
			},
		},
		ReadyShards:    1,
		PendingShards:  0,
		DrainingShards: 0,
		CreatedAt:      fixedTime,
		UpdatedAt:      fixedTime,
	}
}

func sampleSMTPOverview(prefix string) *entity.SMTPOverview {
	return &entity.SMTPOverview{
		Metrics: entity.OverviewMetrics{
			DeliveredToday: 100,
			QueuedNow:      5,
			ActiveGateways: 2,
			TotalGateways:  3,
			LiveTemplates:  4,
			TotalTemplates: 5,
		},
		DeliveryThroughput: []*entity.OverviewThroughputPoint{
			{Label: prefix + "-throughput", Delivered: 12, Queued: 4, Retries: 1},
		},
		HealthDistribution: entity.OverviewHealthDistribution{
			Healthy: 1,
			Warning: 1,
			Stopped: 0,
		},
		QueueMix: []*entity.OverviewQueueMixItem{
			{Category: prefix + "-queue", Pending: 1, Processing: 2, Retries: 3},
		},
		Gateways: []*entity.GatewayListItem{
			{
				ID:                  "gateway-" + prefix,
				Name:                "gateway-" + prefix,
				TrafficClass:        "default",
				Status:              "active",
				RoutingMode:         "direct",
				Priority:            1,
				DesiredShardCount:   1,
				TemplateCount:       1,
				EndpointCount:       1,
				ReadyShards:         1,
				PendingShards:       0,
				DrainingShards:      0,
				FallbackGatewayName: "fallback",
				UpdatedAt:           fixedTime,
			},
		},
		Timeline: []*entity.OverviewTimelineItem{
			{
				ID:         prefix + "-timeline",
				EntityType: "gateway",
				EntityName: "gateway-" + prefix,
				Action:     "updated",
				ActorName:  "actor",
				Note:       "note",
				CreatedAt:  fixedTime,
			},
		},
	}
}

var fixedTime = time.Unix(1700000000, 0).UTC()

type stubTemplatePoolService struct {
	listItems []*entity.TemplateListItem
}

func (s *stubTemplatePoolService) ListTemplateItems(ctx context.Context, workspaceID string) ([]*entity.TemplateListItem, error) {
	return s.listItems, nil
}
func (s *stubTemplatePoolService) GetTemplateDetail(ctx context.Context, workspaceID, templateID string) (*entity.TemplateDetail, error) {
	return &entity.TemplateDetail{}, nil
}
func (s *stubTemplatePoolService) ListTemplates(ctx context.Context, workspaceID string) ([]*entity.Template, error) {
	return nil, nil
}
func (s *stubTemplatePoolService) GetTemplate(ctx context.Context, workspaceID, templateID string) (*entity.Template, error) {
	return nil, nil
}
func (s *stubTemplatePoolService) CreateTemplate(ctx context.Context, template *entity.Template) error {
	return nil
}
func (s *stubTemplatePoolService) UpdateTemplate(ctx context.Context, template *entity.Template) error {
	return nil
}
func (s *stubTemplatePoolService) DeleteTemplate(ctx context.Context, workspaceID, templateID string) error {
	return nil
}

type stubGatewayPoolService struct {
	listItems []*entity.GatewayListItem
	detail    *entity.GatewayDetail
}

func (s *stubGatewayPoolService) ListGatewayItems(ctx context.Context, workspaceID string) ([]*entity.GatewayListItem, error) {
	return s.listItems, nil
}
func (s *stubGatewayPoolService) GetGatewayDetail(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.detail, nil
}
func (s *stubGatewayPoolService) ListGateways(ctx context.Context, workspaceID string) ([]*entity.Gateway, error) {
	return nil, nil
}
func (s *stubGatewayPoolService) GetGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.Gateway, error) {
	return nil, nil
}
func (s *stubGatewayPoolService) UpdateGatewayTemplates(ctx context.Context, workspaceID, gatewayID string, templateIDs []string) (*entity.GatewayDetail, error) {
	return s.detail, nil
}
func (s *stubGatewayPoolService) UpdateGatewayEndpoints(ctx context.Context, workspaceID, gatewayID string, endpointIDs []string) (*entity.GatewayDetail, error) {
	return s.detail, nil
}
func (s *stubGatewayPoolService) StartGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.detail, nil
}
func (s *stubGatewayPoolService) DrainGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.detail, nil
}
func (s *stubGatewayPoolService) DisableGateway(ctx context.Context, workspaceID, gatewayID string) (*entity.GatewayDetail, error) {
	return s.detail, nil
}
func (s *stubGatewayPoolService) CreateGateway(ctx context.Context, gateway *entity.Gateway) error {
	return nil
}
func (s *stubGatewayPoolService) UpdateGateway(ctx context.Context, gateway *entity.Gateway) error {
	return nil
}
func (s *stubGatewayPoolService) DeleteGateway(ctx context.Context, workspaceID, gatewayID string) error {
	return nil
}

type stubAggregationPoolService struct {
	item *entity.SMTPOverview
}

func (s *stubAggregationPoolService) GetWorkspaceAggregation(ctx context.Context, workspaceID string) (*entity.SMTPOverview, error) {
	return s.item, nil
}
