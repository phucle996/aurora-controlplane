CREATE TABLE IF NOT EXISTS core.secret_key_versions (
  id VARCHAR(26) PRIMARY KEY,
  family VARCHAR(32) NOT NULL CHECK (family IN ('access', 'refresh', 'one_time', 'admin_api')),
  version BIGINT NOT NULL CHECK (version > 0),
  state VARCHAR(16) NOT NULL CHECK (state IN ('active', 'previous')),
  secret_ciphertext TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  rotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (family, version),
  UNIQUE (family, state)
);

CREATE INDEX IF NOT EXISTS idx_core_secret_key_versions_family_state
  ON core.secret_key_versions(family, state);

CREATE INDEX IF NOT EXISTS idx_core_secret_key_versions_family_rotated_at
  ON core.secret_key_versions(family, rotated_at DESC);

DROP TRIGGER IF EXISTS set_timestamp_secret_key_versions ON core.secret_key_versions;
CREATE TRIGGER set_timestamp_secret_key_versions
BEFORE UPDATE ON core.secret_key_versions
FOR EACH ROW
EXECUTE FUNCTION core.trigger_set_timestamp();
