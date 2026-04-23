package entity

import "time"

type OverviewMetrics struct {
	DeliveredToday int64
	QueuedNow      int64
	ActiveGateways int64
	TotalGateways  int64
	LiveTemplates  int64
	TotalTemplates int64
}

type OverviewThroughputPoint struct {
	Label     string
	Delivered int64
	Queued    int64
	Retries   int64
}

type OverviewHealthDistribution struct {
	Healthy int64
	Warning int64
	Stopped int64
}

type OverviewQueueMixItem struct {
	Category   string
	Pending    int64
	Processing int64
	Retries    int64
}

type OverviewTimelineItem struct {
	ID         string
	EntityType string
	EntityName string
	Action     string
	ActorName  string
	Note       string
	CreatedAt  time.Time
}

type SMTPOverview struct {
	Metrics            OverviewMetrics
	DeliveryThroughput []*OverviewThroughputPoint
	HealthDistribution OverviewHealthDistribution
	QueueMix           []*OverviewQueueMixItem
	Gateways           []*GatewayListItem
	Timeline           []*OverviewTimelineItem
}

type ConsumerOption struct {
	ID     string
	Label  string
	Status string
}

type TemplateListItem struct {
	ID           string
	Name         string
	Category     string
	TrafficClass string
	Subject      string
	FromEmail    string
	ToEmail      string
	Status       string
	ConsumerID   string
	ConsumerName string
	UpdatedAt    time.Time
}

type TemplateDetail struct {
	ID             string
	Name           string
	Category       string
	TrafficClass   string
	Subject        string
	FromEmail      string
	ToEmail        string
	Status         string
	Variables      []string
	ConsumerID     string
	ConsumerName   string
	TextBody       string
	HTMLBody       string
	ActiveVersion  int
	RuntimeVersion int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type GatewayListItem struct {
	ID                  string
	Name                string
	TrafficClass        string
	Status              string
	RoutingMode         string
	Priority            int
	DesiredShardCount   int
	TemplateCount       int
	EndpointCount       int
	ReadyShards         int
	PendingShards       int
	DrainingShards      int
	FallbackGatewayName string
	UpdatedAt           time.Time
}

type GatewayFallbackSummary struct {
	ID     string
	Name   string
	Status string
}

type GatewayTemplateBinding struct {
	ID           string
	Name         string
	Category     string
	TrafficClass string
	Status       string
	ConsumerID   string
	ConsumerName string
	Selected     bool
	Position     int
}

type GatewayEndpointBinding struct {
	ID       string
	Name     string
	Host     string
	Port     int
	Username string
	Status   string
	Selected bool
	Position int
}

type GatewayDetail struct {
	ID                string
	Name              string
	TrafficClass      string
	Status            string
	RoutingMode       string
	Priority          int
	DesiredShardCount int
	RuntimeVersion    int64
	FallbackGateway   *GatewayFallbackSummary
	Templates         []*GatewayTemplateBinding
	Endpoints         []*GatewayEndpointBinding
	ReadyShards       int
	PendingShards     int
	DrainingShards    int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ConsumerView struct {
	ID                string
	ZoneID            string
	Name              string
	TransportType     string
	Source            string
	ConsumerGroup     string
	WorkerConcurrency int
	AckTimeoutSeconds int
	BatchSize         int
	Status            string
	Note              string
	ConnectionConfig  []byte
	DesiredShardCount int
	HasSecret         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type EndpointView struct {
	ID                   string
	Name                 string
	ProviderKind         string
	Host                 string
	Port                 int
	Username             string
	Priority             int
	Weight               int
	MaxConnections       int
	MaxParallelSends     int
	MaxMessagesPerSecond int
	Burst                int
	WarmupState          string
	Status               string
	TLSMode              string
	HasSecret            bool
	HasCACert            bool
	HasClientCert        bool
	HasClientKey         bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
