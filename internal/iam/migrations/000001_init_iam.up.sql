CREATE SCHEMA IF NOT EXISTS iam;

CREATE TABLE iam.users (
    id VARCHAR(26) PRIMARY KEY,
    username TEXT NOT NULL UNIQUE CHECK (username <> ''),
    email TEXT NOT NULL UNIQUE CHECK (email <> ''),
    full_name TEXT NOT NULL CHECK (full_name <> ''),
    phone_number_ciphertext TEXT,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending_email_verification', 'active', 'locked')),
    authz_version BIGINT NOT NULL DEFAULT 1,
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.mfa_totp_credentials (
    user_id VARCHAR(26) PRIMARY KEY REFERENCES iam.users(id) ON DELETE RESTRICT,
    secret_ciphertext TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.mfa_backup_codes (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.roles (
    id VARCHAR(26) PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.permissions (
    id VARCHAR(26) PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.role_permissions (
    role_id VARCHAR(26) NOT NULL REFERENCES iam.roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(26) NOT NULL REFERENCES iam.permissions(id) ON DELETE RESTRICT,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE iam.user_roles (
    user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    role_id VARCHAR(26) NOT NULL REFERENCES iam.roles(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE iam.devices (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    device_label TEXT,
    fingerprint_hash TEXT NOT NULL,
    proof_key_thumbprint TEXT,
    trusted BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.sessions (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE RESTRICT,
    device_id VARCHAR(26) NOT NULL REFERENCES iam.devices(id) ON DELETE CASCADE,
    auth_method TEXT NOT NULL CHECK (auth_method IN ('password', 'password_mfa', 'recovery_code')),
    mfa_verified BOOLEAN NOT NULL DEFAULT false,
    proof_key_thumbprint TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE iam.refresh_tokens (
    id VARCHAR(26) PRIMARY KEY,
    session_id VARCHAR(26) NOT NULL REFERENCES iam.sessions(id) ON DELETE CASCADE,
    parent_token_id VARCHAR(26) REFERENCES iam.refresh_tokens(id) ON DELETE SET NULL,
    token_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'rotated')),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);
