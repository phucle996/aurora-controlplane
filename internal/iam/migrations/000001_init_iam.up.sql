CREATE SCHEMA IF NOT EXISTS iam;

CREATE OR REPLACE FUNCTION iam.trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION iam.trigger_trim_audit_logs()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM iam.audit_logs
	WHERE created_at < NOW() - INTERVAL '30 days';
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS iam.users (
	id VARCHAR(26) PRIMARY KEY,
	username VARCHAR(255) UNIQUE NOT NULL,
	email VARCHAR(255) UNIQUE NOT NULL,
	phone VARCHAR(20) UNIQUE,
	password_hash TEXT NOT NULL,
	security_level SMALLINT NOT NULL CHECK (security_level >= 0),
	status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'disable')),
	status_reason VARCHAR(255),
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_timestamp_users ON iam.users;
CREATE TRIGGER set_timestamp_users
BEFORE UPDATE ON iam.users
FOR EACH ROW
EXECUTE FUNCTION iam.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS iam.password_histories (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.user_profiles (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) UNIQUE NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	fullname VARCHAR(100),
	avatar_url TEXT,
	bio TEXT,
	timezone VARCHAR(50) DEFAULT 'UTC',
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_timestamp_user_profiles ON iam.user_profiles;
CREATE TRIGGER set_timestamp_user_profiles
BEFORE UPDATE ON iam.user_profiles
FOR EACH ROW
EXECUTE FUNCTION iam.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS iam.devices (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	device_public_key TEXT NOT NULL,
	key_algorithm VARCHAR(20) DEFAULT 'ES256',
	fingerprint TEXT NOT NULL,
	device_name VARCHAR(100),
	last_ip INET,
	last_active_at TIMESTAMPTZ DEFAULT NOW(),
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.refresh_tokens (
	id VARCHAR(26) PRIMARY KEY,
	device_id VARCHAR(26) NOT NULL REFERENCES iam.devices(id) ON DELETE CASCADE,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL UNIQUE,
	expires_at TIMESTAMPTZ NOT NULL,
	is_revoked BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.webauthn_credentials (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	credential_id TEXT UNIQUE NOT NULL,
	public_key TEXT NOT NULL,
	sign_count BIGINT DEFAULT 0,
	device_name VARCHAR(100),
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.mfa_settings (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	mfa_type VARCHAR(50) NOT NULL,
	device_name VARCHAR(100),
	is_primary BOOLEAN DEFAULT FALSE,
	secret_encrypted TEXT NOT NULL,
	is_enabled BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_timestamp_mfa_settings ON iam.mfa_settings;
CREATE TRIGGER set_timestamp_mfa_settings
BEFORE UPDATE ON iam.mfa_settings
FOR EACH ROW
EXECUTE FUNCTION iam.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS iam.recovery_codes (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	code_hash TEXT NOT NULL,
	is_used BOOLEAN DEFAULT FALSE,
	used_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.roles (
	id VARCHAR(26) PRIMARY KEY,
	name VARCHAR(100) UNIQUE NOT NULL,
	description TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.permissions (
	id VARCHAR(26) PRIMARY KEY,
	slug VARCHAR(100) UNIQUE NOT NULL,
	description TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS iam.role_permissions (
	role_id VARCHAR(26) NOT NULL REFERENCES iam.roles(id) ON DELETE CASCADE,
	permission_id VARCHAR(26) NOT NULL REFERENCES iam.permissions(id) ON DELETE CASCADE,
	PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS iam.user_roles (
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	role_id VARCHAR(26) NOT NULL REFERENCES iam.roles(id) ON DELETE CASCADE,
	PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS iam.oauth_clients (
	id VARCHAR(26) PRIMARY KEY,
	client_id VARCHAR(100) UNIQUE NOT NULL,
	client_secret_hash TEXT NOT NULL,
	name VARCHAR(100) NOT NULL,
	redirect_uris JSONB NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_timestamp_oauth_clients ON iam.oauth_clients;
CREATE TRIGGER set_timestamp_oauth_clients
BEFORE UPDATE ON iam.oauth_clients
FOR EACH ROW
EXECUTE FUNCTION iam.trigger_set_timestamp();

CREATE TABLE IF NOT EXISTS iam.oauth_grants (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	client_id VARCHAR(26) NOT NULL REFERENCES iam.oauth_clients(id) ON DELETE CASCADE,
	scopes JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(user_id, client_id)
);

CREATE TABLE IF NOT EXISTS iam.audit_logs (
	id VARCHAR(26) PRIMARY KEY,
	user_id VARCHAR(26) REFERENCES iam.users(id) ON DELETE SET NULL,
	action VARCHAR(100) NOT NULL,
	risk_level SMALLINT DEFAULT 1,
	ip_address INET,
	user_agent TEXT,
	device_id VARCHAR(26),
	metadata JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS trim_audit_logs_30_days ON iam.audit_logs;
CREATE TRIGGER trim_audit_logs_30_days
AFTER INSERT ON iam.audit_logs
FOR EACH ROW
EXECUTE FUNCTION iam.trigger_trim_audit_logs();
