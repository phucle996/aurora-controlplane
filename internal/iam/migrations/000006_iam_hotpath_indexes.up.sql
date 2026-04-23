CREATE INDEX IF NOT EXISTS idx_iam_refresh_tokens_user_id ON iam.refresh_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_iam_refresh_tokens_expires_at ON iam.refresh_tokens(expires_at);

CREATE INDEX IF NOT EXISTS idx_iam_devices_user_id_fingerprint_last_active_at
	ON iam.devices(user_id, fingerprint, last_active_at DESC);
