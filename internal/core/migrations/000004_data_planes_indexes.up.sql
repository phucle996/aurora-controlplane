CREATE UNIQUE INDEX IF NOT EXISTS idx_core_data_planes_node_key ON core.data_planes(node_key);
CREATE INDEX IF NOT EXISTS idx_core_data_planes_status ON core.data_planes(status);
CREATE INDEX IF NOT EXISTS idx_core_data_planes_last_seen_at ON core.data_planes(last_seen_at);
