CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_consumers_workspace_name ON smtp.consumers (workspace_id, LOWER(name)) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_templates_workspace_name ON smtp.templates (workspace_id, LOWER(name)) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_gateways_workspace_name ON smtp.gateways (workspace_id, LOWER(name)) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_endpoints_workspace_name ON smtp.endpoints (workspace_id, LOWER(name)) WHERE workspace_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_smtp_consumers_workspace_created_at ON smtp.consumers (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_templates_workspace_created_at ON smtp.templates (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_gateways_workspace_created_at ON smtp.gateways (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_endpoints_workspace_created_at ON smtp.endpoints (workspace_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_smtp_consumers_zone_id ON smtp.consumers (zone_id);
CREATE INDEX IF NOT EXISTS idx_smtp_gateways_zone_id ON smtp.gateways (zone_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_templates_live_consumer ON smtp.templates (consumer_id) WHERE status = 'live' AND consumer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_smtp_templates_traffic_class ON smtp.templates (traffic_class);
CREATE INDEX IF NOT EXISTS idx_smtp_gateways_traffic_class_priority ON smtp.gateways (traffic_class, priority, created_at);
CREATE INDEX IF NOT EXISTS idx_smtp_gateways_fallback_gateway_id ON smtp.gateways (fallback_gateway_id);
CREATE INDEX IF NOT EXISTS idx_smtp_gateway_endpoints_gateway_pos ON smtp.gateway_endpoints (gateway_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_gateway_templates_template_id ON smtp.gateway_templates (template_id);
CREATE INDEX IF NOT EXISTS idx_smtp_gateway_templates_gateway_position ON smtp.gateway_templates (gateway_id, position, template_id);

CREATE INDEX IF NOT EXISTS idx_smtp_delivery_attempts_workspace_created_at ON smtp.delivery_attempts (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_activity_logs_workspace_created_at ON smtp.activity_logs (workspace_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_smtp_consumer_shards_desired_state ON smtp.consumer_shards (desired_state, consumer_id, shard_id);
CREATE INDEX IF NOT EXISTS idx_smtp_gateway_shards_desired_state ON smtp.gateway_shards (desired_state, gateway_id, shard_id);

CREATE INDEX IF NOT EXISTS idx_smtp_runtime_heartbeats_updated_at ON smtp.runtime_heartbeats (updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_gateway_runtime_statuses_report_at ON smtp.gateway_runtime_statuses (last_report_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_consumer_runtime_statuses_report_at ON smtp.consumer_runtime_statuses (last_report_at DESC);

CREATE INDEX IF NOT EXISTS idx_smtp_consumer_assignments_data_plane_lease ON smtp.consumer_assignments (data_plane_id, lease_expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_smtp_gateway_shard_assignments_data_plane_lease ON smtp.gateway_shard_assignments (data_plane_id, lease_expires_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_consumer_assignments_active_consumer_shard ON smtp.consumer_assignments (consumer_id, shard_id) WHERE assignment_state = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS idx_smtp_gateway_shard_assignments_active_shard ON smtp.gateway_shard_assignments (gateway_id, shard_id) WHERE assignment_state = 'active';
CREATE INDEX IF NOT EXISTS idx_smtp_consumer_assignments_generation ON smtp.consumer_assignments (generation DESC, assignment_state);
CREATE INDEX IF NOT EXISTS idx_smtp_gateway_shard_assignments_generation ON smtp.gateway_shard_assignments (generation DESC, assignment_state);
