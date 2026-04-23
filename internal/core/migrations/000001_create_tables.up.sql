CREATE SCHEMA IF NOT EXISTS core;

CREATE OR REPLACE FUNCTION core.trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS core.tenants (
  id VARCHAR(26) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(100) UNIQUE NOT NULL,
  status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'archived')),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_timestamp_tenants ON core.tenants;
CREATE TRIGGER set_timestamp_tenants
BEFORE UPDATE ON core.tenants
FOR EACH ROW
EXECUTE FUNCTION core.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS core.tenant_members (
  tenant_id VARCHAR(26) REFERENCES core.tenants(id) ON DELETE CASCADE,
  user_id VARCHAR(26) NOT NULL,
  role_id VARCHAR(26) NOT NULL,
  joined_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS core.zones (
  id VARCHAR(26) PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description TEXT,
  slug VARCHAR(100),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

WITH base_slugs AS (
  SELECT
    id,
    COALESCE(
      NULLIF(
        TRIM(BOTH '-' FROM regexp_replace(lower(trim(name)), '[^a-z0-9]+', '-', 'g')),
        ''
      ),
      'zone'
    ) AS base_slug
  FROM core.zones
),
ranked_slugs AS (
  SELECT
    id,
    base_slug,
    ROW_NUMBER() OVER (PARTITION BY base_slug ORDER BY id) AS slug_rank
  FROM base_slugs
)
UPDATE core.zones z
SET slug = CASE
  WHEN r.slug_rank = 1 THEN r.base_slug
  ELSE left(r.base_slug, 90) || '-' || lower(substr(z.id, 1, 8))
END
FROM ranked_slugs r
WHERE z.id = r.id
  AND COALESCE(z.slug, '') = '';



CREATE TABLE IF NOT EXISTS core.data_planes (
  id VARCHAR(26) PRIMARY KEY,
  node_key VARCHAR(128) UNIQUE NOT NULL,
  name VARCHAR(100) NOT NULL,
  zone_id VARCHAR(26) REFERENCES core.zones(id) ON DELETE SET NULL,
  grpc_endpoint TEXT NOT NULL,
  version TEXT NOT NULL DEFAULT '',
  cert_serial TEXT NOT NULL DEFAULT '',
  cert_not_after TIMESTAMPTZ,
  status VARCHAR(20) DEFAULT 'healthy',
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_timestamp_data_planes ON core.data_planes;
CREATE TRIGGER set_timestamp_data_planes
BEFORE UPDATE ON core.data_planes
FOR EACH ROW
EXECUTE FUNCTION core.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS core.workspaces (
  id VARCHAR(26) PRIMARY KEY,
  tenant_id VARCHAR(26) REFERENCES core.tenants(id) ON DELETE CASCADE,
  data_plane_id VARCHAR(26) REFERENCES core.data_planes(id),
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(100) NOT NULL,
  status VARCHAR(20) DEFAULT 'active',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, slug)
);

DROP TRIGGER IF EXISTS set_timestamp_workspaces ON core.workspaces;
CREATE TRIGGER set_timestamp_workspaces
BEFORE UPDATE ON core.workspaces
FOR EACH ROW
EXECUTE FUNCTION core.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS core.workspace_members (
  workspace_id VARCHAR(26) REFERENCES core.workspaces(id) ON DELETE CASCADE,
  user_id VARCHAR(26) NOT NULL,
  role_id VARCHAR(26) NOT NULL,
  joined_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (workspace_id, user_id)
);
