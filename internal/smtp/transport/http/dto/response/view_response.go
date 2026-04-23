package smtp_resdto

import (
	"encoding/json"
	"time"
)

type OverviewMetrics struct {
	DeliveredToday int64 `json:"delivered_today"`
	QueuedNow      int64 `json:"queued_now"`
	ActiveGateways int64 `json:"active_gateways"`
	TotalGateways  int64 `json:"total_gateways"`
	LiveTemplates  int64 `json:"live_templates"`
	TotalTemplates int64 `json:"total_templates"`
}

type OverviewThroughputPoint struct {
	Label     string `json:"label"`
	Delivered int64  `json:"delivered"`
	Queued    int64  `json:"queued"`
	Retries   int64  `json:"retries"`
}

type OverviewHealthDistribution struct {
	Healthy int64 `json:"healthy"`
	Warning int64 `json:"warning"`
	Stopped int64 `json:"stopped"`
}

type OverviewQueueMixItem struct {
	Category   string `json:"category"`
	Pending    int64  `json:"pending"`
	Processing int64  `json:"processing"`
	Retries    int64  `json:"retries"`
}

type OverviewTimelineItem struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityName string    `json:"entity_name"`
	Action     string    `json:"action"`
	ActorName  string    `json:"actor_name"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type SMTPOverview struct {
	Metrics            OverviewMetrics            `json:"metrics"`
	DeliveryThroughput []*OverviewThroughputPoint `json:"delivery_throughput"`
	HealthDistribution OverviewHealthDistribution `json:"health_distribution"`
	QueueMix           []*OverviewQueueMixItem    `json:"queue_mix"`
	Gateways           []*GatewayListItem         `json:"gateways"`
	Timeline           []*OverviewTimelineItem    `json:"timeline"`
}

type ConsumerOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
}

type TemplateListItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	TrafficClass string    `json:"traffic_class"`
	Subject      string    `json:"subject"`
	FromEmail    string    `json:"from_email"`
	ToEmail      string    `json:"to_email"`
	Status       string    `json:"status"`
	ConsumerID   string    `json:"consumer_id"`
	ConsumerName string    `json:"consumer_name"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TemplateDetail struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	TrafficClass   string    `json:"traffic_class"`
	Subject        string    `json:"subject"`
	FromEmail      string    `json:"from_email"`
	ToEmail        string    `json:"to_email"`
	Status         string    `json:"status"`
	Variables      []string  `json:"variables"`
	ConsumerID     string    `json:"consumer_id"`
	ConsumerName   string    `json:"consumer_name"`
	TextBody       string    `json:"text_body"`
	HTMLBody       string    `json:"html_body"`
	ActiveVersion  int       `json:"active_version"`
	RuntimeVersion int64     `json:"runtime_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GatewayListItem struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	TrafficClass        string    `json:"traffic_class"`
	Status              string    `json:"status"`
	RoutingMode         string    `json:"routing_mode"`
	Priority            int       `json:"priority"`
	DesiredShardCount   int       `json:"desired_shard_count"`
	TemplateCount       int       `json:"template_count"`
	EndpointCount       int       `json:"endpoint_count"`
	ReadyShards         int       `json:"ready_shards"`
	PendingShards       int       `json:"pending_shards"`
	DrainingShards      int       `json:"draining_shards"`
	FallbackGatewayName string    `json:"fallback_gateway_name"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type GatewayFallbackSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type GatewayTemplateBinding struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	TrafficClass string `json:"traffic_class"`
	Status       string `json:"status"`
	ConsumerID   string `json:"consumer_id"`
	ConsumerName string `json:"consumer_name"`
	Selected     bool   `json:"selected"`
	Position     int    `json:"position"`
}

type GatewayEndpointBinding struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Status   string `json:"status"`
	Selected bool   `json:"selected"`
	Position int    `json:"position"`
}

type GatewayDetail struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name"`
	TrafficClass      string                    `json:"traffic_class"`
	Status            string                    `json:"status"`
	RoutingMode       string                    `json:"routing_mode"`
	Priority          int                       `json:"priority"`
	DesiredShardCount int                       `json:"desired_shard_count"`
	RuntimeVersion    int64                     `json:"runtime_version"`
	FallbackGateway   *GatewayFallbackSummary   `json:"fallback_gateway"`
	Templates         []*GatewayTemplateBinding `json:"templates"`
	Endpoints         []*GatewayEndpointBinding `json:"endpoints"`
	ReadyShards       int                       `json:"ready_shards"`
	PendingShards     int                       `json:"pending_shards"`
	DrainingShards    int                       `json:"draining_shards"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
}

type ConsumerView struct {
	ID                string          `json:"id"`
	ZoneID            string          `json:"zone_id"`
	Name              string          `json:"name"`
	TransportType     string          `json:"transport_type"`
	Source            string          `json:"source"`
	ConsumerGroup     string          `json:"consumer_group"`
	WorkerConcurrency int             `json:"worker_concurrency"`
	AckTimeoutSeconds int             `json:"ack_timeout_seconds"`
	BatchSize         int             `json:"batch_size"`
	Status            string          `json:"status"`
	Note              string          `json:"note"`
	ConnectionConfig  json.RawMessage `json:"connection_config"`
	DesiredShardCount int             `json:"desired_shard_count"`
	HasSecret         bool            `json:"has_secret"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type EndpointView struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	ProviderKind         string    `json:"provider_kind"`
	Host                 string    `json:"host"`
	Port                 int       `json:"port"`
	Username             string    `json:"username"`
	Priority             int       `json:"priority"`
	Weight               int       `json:"weight"`
	MaxConnections       int       `json:"max_connections"`
	MaxParallelSends     int       `json:"max_parallel_sends"`
	MaxMessagesPerSecond int       `json:"max_messages_per_second"`
	Burst                int       `json:"burst"`
	WarmupState          string    `json:"warmup_state"`
	Status               string    `json:"status"`
	TLSMode              string    `json:"tls_mode"`
	HasSecret            bool      `json:"has_secret"`
	HasCACert            bool      `json:"has_ca_cert"`
	HasClientCert        bool      `json:"has_client_cert"`
	HasClientKey         bool      `json:"has_client_key"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
