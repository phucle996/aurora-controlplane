CREATE SEQUENCE IF NOT EXISTS smtp.runtime_version_seq AS BIGINT START WITH 1000;

CREATE TABLE IF NOT EXISTS smtp.consumers (
  id VARCHAR(26) PRIMARY KEY,
  owner_user_id VARCHAR(26) NULL REFERENCES iam.users(id) ON DELETE CASCADE,
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  zone_id VARCHAR(26) REFERENCES core.zones(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  transport_type smtp.consumer_transport_type NOT NULL,
  source TEXT NOT NULL,
  consumer_group TEXT NOT NULL,
  worker_concurrency INT NOT NULL DEFAULT 1,
  ack_timeout_seconds INT NOT NULL DEFAULT 30,
  batch_size INT NOT NULL DEFAULT 128,
  status smtp.consumer_status NOT NULL DEFAULT 'disabled',
  note TEXT NOT NULL DEFAULT '',
  lag BIGINT NOT NULL DEFAULT 0,
  connection_config JSONB NOT NULL DEFAULT '{}'::jsonb,
  runtime_version BIGINT NOT NULL DEFAULT nextval('smtp.runtime_version_seq'),
  desired_shard_count INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.consumer_secrets (
  consumer_id VARCHAR(26) PRIMARY KEY REFERENCES smtp.consumers(id) ON DELETE CASCADE,
  secret_config JSONB NOT NULL DEFAULT '{}'::jsonb,
  secret_ref TEXT NOT NULL DEFAULT '',
  secret_version BIGINT NOT NULL DEFAULT 1,
  provider TEXT NOT NULL DEFAULT 'postgresql',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.gateways (
  id VARCHAR(26) PRIMARY KEY,
  owner_user_id VARCHAR(26) NULL REFERENCES iam.users(id) ON DELETE CASCADE,
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  zone_id VARCHAR(26) REFERENCES core.zones(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  traffic_class TEXT NOT NULL DEFAULT 'transactional',
  status smtp.gateway_status NOT NULL DEFAULT 'disabled',
  routing_mode TEXT NOT NULL DEFAULT 'round_robin',
  priority INT NOT NULL DEFAULT 100,
  fallback_gateway_id VARCHAR(26) NULL REFERENCES smtp.gateways(id) ON DELETE SET NULL,
  runtime_version BIGINT NOT NULL DEFAULT nextval('smtp.runtime_version_seq'),
  desired_shard_count INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.templates (
  id VARCHAR(26) PRIMARY KEY,
  owner_user_id VARCHAR(26) NULL REFERENCES iam.users(id) ON DELETE CASCADE,
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  traffic_class TEXT NOT NULL DEFAULT 'transactional',
  subject TEXT NOT NULL,
  from_email TEXT NOT NULL,
  to_email TEXT NOT NULL,
  status smtp.template_status NOT NULL DEFAULT 'draft',
  variables TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  consumer_id VARCHAR(26) NULL REFERENCES smtp.consumers(id) ON DELETE SET NULL,
  active_version INT NOT NULL DEFAULT 1,
  retry_max_attempts INT NOT NULL DEFAULT 3,
  retry_backoff_seconds INT NOT NULL DEFAULT 5,
  text_body TEXT NOT NULL,
  html_body TEXT NOT NULL DEFAULT '',
  runtime_version BIGINT NOT NULL DEFAULT nextval('smtp.runtime_version_seq'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.template_versions (
  template_id VARCHAR(26) NOT NULL REFERENCES smtp.templates(id) ON DELETE CASCADE,
  version INT NOT NULL,
  subject TEXT NOT NULL,
  from_email TEXT NOT NULL,
  to_email TEXT NOT NULL,
  variables TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  text_body TEXT NOT NULL,
  html_body TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (template_id, version)
);

CREATE TABLE IF NOT EXISTS smtp.endpoints (
  id VARCHAR(26) PRIMARY KEY,
  owner_user_id VARCHAR(26) NULL REFERENCES iam.users(id) ON DELETE CASCADE,
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  provider_kind TEXT NOT NULL DEFAULT 'smtp',
  host TEXT NOT NULL,
  port INT NOT NULL,
  username TEXT NOT NULL DEFAULT '',
  priority INT NOT NULL DEFAULT 100,
  weight INT NOT NULL DEFAULT 1,
  max_connections INT NOT NULL DEFAULT 16,
  max_parallel_sends INT NOT NULL DEFAULT 16,
  max_messages_per_second INT NOT NULL DEFAULT 0,
  burst INT NOT NULL DEFAULT 0,
  warmup_state TEXT NOT NULL DEFAULT 'stable',
  status smtp.endpoint_status NOT NULL DEFAULT 'disabled',
  tls_mode smtp.tls_mode NOT NULL DEFAULT 'starttls',
  runtime_version BIGINT NOT NULL DEFAULT nextval('smtp.runtime_version_seq'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.endpoint_secrets (
  endpoint_id VARCHAR(26) PRIMARY KEY REFERENCES smtp.endpoints(id) ON DELETE CASCADE,
  password TEXT NOT NULL DEFAULT '',
  ca_cert_pem TEXT NOT NULL DEFAULT '',
  client_cert_pem TEXT NOT NULL DEFAULT '',
  client_key_pem TEXT NOT NULL DEFAULT '',
  secret_ref TEXT NOT NULL DEFAULT '',
  secret_version BIGINT NOT NULL DEFAULT 1,
  provider TEXT NOT NULL DEFAULT 'postgresql',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.gateway_endpoints (
  gateway_id VARCHAR(26) NOT NULL REFERENCES smtp.gateways(id) ON DELETE CASCADE,
  endpoint_id VARCHAR(26) NOT NULL REFERENCES smtp.endpoints(id) ON DELETE CASCADE,
  position INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (gateway_id, endpoint_id)
);

CREATE TABLE IF NOT EXISTS smtp.gateway_templates (
  gateway_id VARCHAR(26) NOT NULL REFERENCES smtp.gateways(id) ON DELETE CASCADE,
  template_id VARCHAR(26) NOT NULL REFERENCES smtp.templates(id) ON DELETE CASCADE,
  position INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (gateway_id, template_id)
);

CREATE TABLE IF NOT EXISTS smtp.activity_logs (
  id VARCHAR(26) PRIMARY KEY,
  entity_type TEXT NOT NULL,
  entity_id VARCHAR(26) NOT NULL,
  entity_name TEXT NOT NULL,
  action TEXT NOT NULL,
  actor_name TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.delivery_attempts (
  id VARCHAR(26) PRIMARY KEY,
  consumer_id VARCHAR(26) NULL REFERENCES smtp.consumers(id) ON DELETE SET NULL,
  template_id VARCHAR(26) NULL REFERENCES smtp.templates(id) ON DELETE SET NULL,
  gateway_id VARCHAR(26) NULL REFERENCES smtp.gateways(id) ON DELETE SET NULL,
  endpoint_id VARCHAR(26) NULL REFERENCES smtp.endpoints(id) ON DELETE SET NULL,
  message_id TEXT NOT NULL DEFAULT '',
  transport_message_id TEXT NOT NULL DEFAULT '',
  subject TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  error_message TEXT NOT NULL DEFAULT '',
  error_class TEXT NOT NULL DEFAULT '',
  retry_count INT NOT NULL DEFAULT 0,
  trace_id TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.consumer_shards (
  consumer_id VARCHAR(26) NOT NULL REFERENCES smtp.consumers(id) ON DELETE CASCADE,
  shard_id INT NOT NULL,
  desired_state TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (consumer_id, shard_id)
);

CREATE TABLE IF NOT EXISTS smtp.gateway_shards (
  gateway_id VARCHAR(26) NOT NULL REFERENCES smtp.gateways(id) ON DELETE CASCADE,
  shard_id INT NOT NULL,
  desired_state TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (gateway_id, shard_id)
);

CREATE TABLE IF NOT EXISTS smtp.runtime_heartbeats (
  data_plane_id VARCHAR(26) PRIMARY KEY REFERENCES core.data_planes(id) ON DELETE CASCADE,
  sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  local_version BIGINT NOT NULL DEFAULT 0,
  gateway_count INT NOT NULL DEFAULT 0,
  consumer_count INT NOT NULL DEFAULT 0,
  member_state TEXT NOT NULL DEFAULT 'joining',
  capacity INT NOT NULL DEFAULT 1,
  grpc_addr TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS smtp.gateway_runtime_statuses (
  gateway_id VARCHAR(26) NOT NULL REFERENCES smtp.gateways(id) ON DELETE CASCADE,
  shard_id INT NOT NULL DEFAULT 0,
  data_plane_id VARCHAR(26) NOT NULL REFERENCES core.data_planes(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  inflight_count BIGINT NOT NULL DEFAULT 0,
  consumer_count INT NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  version BIGINT NOT NULL DEFAULT 0,
  generation BIGINT NOT NULL DEFAULT 0,
  assignment_state TEXT NOT NULL DEFAULT 'active',
  revoking_done BOOLEAN NOT NULL DEFAULT FALSE,
  last_report_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (gateway_id, shard_id, data_plane_id)
);

CREATE TABLE IF NOT EXISTS smtp.consumer_runtime_statuses (
  consumer_id VARCHAR(26) NOT NULL REFERENCES smtp.consumers(id) ON DELETE CASCADE,
  shard_id INT NOT NULL DEFAULT 0,
  data_plane_id VARCHAR(26) NOT NULL REFERENCES core.data_planes(id) ON DELETE CASCADE,
  gateway_id VARCHAR(26) NULL REFERENCES smtp.gateways(id) ON DELETE SET NULL,
  status TEXT NOT NULL,
  inflight_count BIGINT NOT NULL DEFAULT 0,
  worker_count INT NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  version BIGINT NOT NULL DEFAULT 0,
  generation BIGINT NOT NULL DEFAULT 0,
  assignment_state TEXT NOT NULL DEFAULT 'active',
  revoking_done BOOLEAN NOT NULL DEFAULT FALSE,
  last_report_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (consumer_id, shard_id, data_plane_id)
);

CREATE TABLE IF NOT EXISTS smtp.consumer_assignments (
  consumer_id VARCHAR(26) NOT NULL,
  shard_id INT NOT NULL,
  data_plane_id VARCHAR(26) NOT NULL REFERENCES core.data_planes(id) ON DELETE CASCADE,
  generation BIGINT NOT NULL,
  assignment_state TEXT NOT NULL,
  desired_state TEXT NOT NULL,
  lease_expires_at TIMESTAMPTZ NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (consumer_id, shard_id, data_plane_id),
  FOREIGN KEY (consumer_id, shard_id) REFERENCES smtp.consumer_shards(consumer_id, shard_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS smtp.gateway_shard_assignments (
  gateway_id VARCHAR(26) NOT NULL,
  shard_id INT NOT NULL,
  data_plane_id VARCHAR(26) NOT NULL REFERENCES core.data_planes(id) ON DELETE CASCADE,
  generation BIGINT NOT NULL,
  assignment_state TEXT NOT NULL,
  desired_state TEXT NOT NULL,
  lease_expires_at TIMESTAMPTZ NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (gateway_id, shard_id, data_plane_id),
  FOREIGN KEY (gateway_id, shard_id) REFERENCES smtp.gateway_shards(gateway_id, shard_id) ON DELETE CASCADE
);
