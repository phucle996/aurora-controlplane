CREATE SCHEMA IF NOT EXISTS virtual_machine;

CREATE TABLE IF NOT EXISTS virtual_machine.hosts (
    host_id VARCHAR(26) PRIMARY KEY,
    agent_id VARCHAR(26) NOT NULL UNIQUE,
    zone_id VARCHAR(26) NOT NULL REFERENCES core.zones(id) ON DELETE RESTRICT,
    data_plane_id VARCHAR(26) NOT NULL REFERENCES core.data_planes(id) ON DELETE RESTRICT,
    hostname VARCHAR(255) NOT NULL,
    private_ip VARCHAR(64) NOT NULL,
    hypervisor_type VARCHAR(64) NOT NULL,
    agent_version VARCHAR(64) NOT NULL DEFAULT '',
    capabilities_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    cpu_cores INTEGER NOT NULL DEFAULT 0,
    memory_bytes BIGINT NOT NULL DEFAULT 0,
    disk_bytes BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'online'
        CHECK (status IN ('online', 'offline', 'degraded', 'quarantined')),
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_virtual_machine_hosts_zone_status
    ON virtual_machine.hosts(zone_id, status);

CREATE INDEX IF NOT EXISTS idx_virtual_machine_hosts_last_seen_at
    ON virtual_machine.hosts(last_seen_at DESC);

