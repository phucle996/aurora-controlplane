CREATE INDEX IF NOT EXISTS idx_core_tenant_members_user_id ON core.tenant_members(user_id);
CREATE INDEX IF NOT EXISTS idx_core_tenant_members_role_id ON core.tenant_members(role_id);

CREATE INDEX IF NOT EXISTS idx_core_workspaces_tenant_id ON core.workspaces(tenant_id);
CREATE INDEX IF NOT EXISTS idx_core_workspaces_data_plane_id ON core.workspaces(data_plane_id);

CREATE INDEX IF NOT EXISTS idx_core_workspace_members_user_id ON core.workspace_members(user_id);
CREATE INDEX IF NOT EXISTS idx_core_workspace_members_role_id ON core.workspace_members(role_id);
CREATE INDEX IF NOT EXISTS idx_core_data_planes_zone_id ON core.data_planes(zone_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_core_zones_slug ON core.zones(slug);
