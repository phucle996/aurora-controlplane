CREATE TABLE IF NOT EXISTS core.admin_api_tokens (
  id VARCHAR(26) PRIMARY KEY,
  token_hash TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO core.admin_api_tokens (id, token_hash, created_at)
SELECT id, token_hash, created_at
FROM iam.admin_api_tokens
ON CONFLICT (id) DO NOTHING;

DROP TABLE IF EXISTS iam.admin_api_tokens;
