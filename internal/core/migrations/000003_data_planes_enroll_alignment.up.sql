DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'core'
      AND table_name = 'data_planes'
      AND column_name = 'grpc_addr'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'core'
      AND table_name = 'data_planes'
      AND column_name = 'grpc_endpoint'
  ) THEN
    ALTER TABLE core.data_planes RENAME COLUMN grpc_addr TO grpc_endpoint;
  END IF;
END;
$$;

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS node_key VARCHAR(128);

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS region VARCHAR(50) NOT NULL DEFAULT '';

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS version TEXT NOT NULL DEFAULT '';

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS cert_serial TEXT NOT NULL DEFAULT '';

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS cert_not_after TIMESTAMPTZ;

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ;

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE core.data_planes
SET node_key = id
WHERE COALESCE(node_key, '') = '';

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'core'
      AND table_name = 'data_planes'
      AND column_name = 'node_key'
      AND is_nullable = 'YES'
  ) THEN
    ALTER TABLE core.data_planes
      ALTER COLUMN node_key SET NOT NULL;
  END IF;
END;
$$;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS agent_token_hash;

DROP TRIGGER IF EXISTS set_timestamp_data_planes ON core.data_planes;
CREATE TRIGGER set_timestamp_data_planes
BEFORE UPDATE ON core.data_planes
FOR EACH ROW
EXECUTE FUNCTION core.trigger_set_timestamp();
