CREATE UNIQUE INDEX idx_iam_users_username ON iam.users(username);
CREATE UNIQUE INDEX idx_iam_users_email ON iam.users(email);
CREATE INDEX idx_iam_users_status ON iam.users(status);

CREATE INDEX idx_iam_mfa_totp_credentials_enabled ON iam.mfa_totp_credentials(enabled);

CREATE INDEX idx_iam_mfa_backup_codes_user_id ON iam.mfa_backup_codes(user_id);
CREATE INDEX idx_iam_mfa_backup_codes_user_id_used_at ON iam.mfa_backup_codes(user_id, used_at);

CREATE UNIQUE INDEX idx_iam_roles_code ON iam.roles(code);
CREATE INDEX idx_iam_roles_is_system ON iam.roles(is_system);

CREATE UNIQUE INDEX idx_iam_permissions_code ON iam.permissions(code);

CREATE INDEX idx_iam_role_permissions_permission_id ON iam.role_permissions(permission_id);

CREATE INDEX idx_iam_user_roles_role_id ON iam.user_roles(role_id);

CREATE INDEX idx_iam_devices_user_id ON iam.devices(user_id);
CREATE INDEX idx_iam_devices_user_id_last_seen_at ON iam.devices(user_id, last_seen_at DESC);
CREATE INDEX idx_iam_devices_trusted ON iam.devices(trusted);
CREATE INDEX idx_iam_devices_proof_key_thumbprint ON iam.devices(proof_key_thumbprint);

CREATE INDEX idx_iam_sessions_user_id ON iam.sessions(user_id);
CREATE INDEX idx_iam_sessions_device_id ON iam.sessions(device_id);
CREATE INDEX idx_iam_sessions_expires_at ON iam.sessions(expires_at);
CREATE INDEX idx_iam_sessions_user_id_last_seen_at ON iam.sessions(user_id, last_seen_at DESC);

CREATE UNIQUE INDEX idx_iam_refresh_tokens_token_hash ON iam.refresh_tokens(token_hash);
CREATE INDEX idx_iam_refresh_tokens_session_id ON iam.refresh_tokens(session_id);
CREATE INDEX idx_iam_refresh_tokens_session_id_status ON iam.refresh_tokens(session_id, status);
CREATE INDEX idx_iam_refresh_tokens_expires_at ON iam.refresh_tokens(expires_at);
