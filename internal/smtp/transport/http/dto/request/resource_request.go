package smtp_reqdto

import "encoding/json"

type ConsumerRequest struct {
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
	SecretConfig      json.RawMessage `json:"secret_config"`
	SecretRef         string          `json:"secret_ref"`
	SecretProvider    string          `json:"secret_provider"`
}

type TemplateRequest struct {
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	TrafficClass        string   `json:"traffic_class"`
	Subject             string   `json:"subject"`
	FromEmail           string   `json:"from_email"`
	ToEmail             string   `json:"to_email"`
	Status              string   `json:"status"`
	Variables           []string `json:"variables"`
	ConsumerID          string   `json:"consumer_id"`
	RetryMaxAttempts    int      `json:"retry_max_attempts"`
	RetryBackoffSeconds int      `json:"retry_backoff_seconds"`
	TextBody            string   `json:"text_body"`
	HTMLBody            string   `json:"html_body"`
}

type GatewayRequest struct {
	ZoneID            string   `json:"zone_id"`
	Name              string   `json:"name"`
	TrafficClass      string   `json:"traffic_class"`
	Status            string   `json:"status"`
	RoutingMode       string   `json:"routing_mode"`
	Priority          int      `json:"priority"`
	FallbackGatewayID string   `json:"fallback_gateway_id"`
	DesiredShardCount int      `json:"desired_shard_count"`
	TemplateIDs       []string `json:"template_ids"`
	EndpointIDs       []string `json:"endpoint_ids"`
}

type EndpointRequest struct {
	Name                 string  `json:"name"`
	ProviderKind         string  `json:"provider_kind"`
	Host                 string  `json:"host"`
	Port                 int     `json:"port"`
	Username             string  `json:"username"`
	Priority             int     `json:"priority"`
	Weight               int     `json:"weight"`
	MaxConnections       int     `json:"max_connections"`
	MaxParallelSends     int     `json:"max_parallel_sends"`
	MaxMessagesPerSecond int     `json:"max_messages_per_second"`
	Burst                int     `json:"burst"`
	WarmupState          string  `json:"warmup_state"`
	Status               string  `json:"status"`
	TLSMode              string  `json:"tls_mode"`
	Password             string  `json:"password"`
	CACertPEM            *string `json:"ca_cert_pem"`
	ClientCertPEM        *string `json:"client_cert_pem"`
	ClientKeyPEM         *string `json:"client_key_pem"`
	SecretRef            *string `json:"secret_ref"`
	SecretProvider       *string `json:"secret_provider"`
}
