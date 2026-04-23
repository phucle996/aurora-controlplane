CREATE TABLE IF NOT EXISTS iam.admin_api_tokens (
  id VARCHAR(26) PRIMARY KEY,
  token_hash TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
  IF to_regclass('core.admin_api_tokens') IS NOT NULL THEN
    INSERT INTO iam.admin_api_tokens (id, token_hash, created_at)
    SELECT id, token_hash, created_at
    FROM core.admin_api_tokens
    ON CONFLICT (id) DO NOTHING;

    DROP TABLE IF EXISTS core.admin_api_tokens;
  END IF;
END;
$$;
