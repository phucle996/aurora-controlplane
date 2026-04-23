DROP TRIGGER IF EXISTS set_timestamp_data_planes ON core.data_planes;

ALTER TABLE core.data_planes
  ADD COLUMN IF NOT EXISTS agent_token_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS updated_at;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS last_seen_at;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS cert_not_after;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS cert_serial;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS version;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS region;

ALTER TABLE core.data_planes
  DROP COLUMN IF EXISTS node_key;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'core'
      AND table_name = 'data_planes'
      AND column_name = 'grpc_endpoint'
  ) THEN
    ALTER TABLE core.data_planes RENAME COLUMN grpc_endpoint TO grpc_addr;
  END IF;
END;
$$;
