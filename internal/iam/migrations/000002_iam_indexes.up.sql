CREATE INDEX IF NOT EXISTS idx_iam_password_histories_user_id ON iam.password_histories(user_id);

CREATE INDEX IF NOT EXISTS idx_iam_devices_user_id ON iam.devices(user_id);

CREATE INDEX IF NOT EXISTS idx_iam_refresh_tokens_device_id ON iam.refresh_tokens(device_id);

CREATE INDEX IF NOT EXISTS idx_iam_webauthn_credentials_user_id ON iam.webauthn_credentials(user_id);

CREATE INDEX IF NOT EXISTS idx_iam_mfa_settings_user_id ON iam.mfa_settings(user_id);

CREATE INDEX IF NOT EXISTS idx_iam_recovery_codes_user_id ON iam.recovery_codes(user_id);

CREATE INDEX IF NOT EXISTS idx_iam_role_permissions_permission_id ON iam.role_permissions(permission_id);

CREATE INDEX IF NOT EXISTS idx_iam_user_roles_role_id ON iam.user_roles(role_id);

CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_user_id ON iam.audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_action ON iam.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_created_at ON iam.audit_logs(created_at);
