IAM DATABASE DESIGN SPEC
Schema: iam
Database: PostgreSQL
Scope: control plane IAM only

1. Design Goal

This schema must be:
- production-ready
- minimal
- easy to reason about
- easy to implement in code
- strict on data integrity
- not overloaded with debug or speculative columns

This schema is intentionally designed to avoid:
- too many nullable columns
- duplicated normalization columns
- tables that exist only for “maybe later”
- audit/event/history tables inside core IAM
- token tables that belong to other services

2. Core Principles

2.1 One table, one responsibility
- user identity stays in users
- MFA secrets stay in MFA tables
- RBAC stays in RBAC tables
- runtime auth state stays in device/session/refresh tables

2.2 Store canonical values directly
- username is stored already canonicalized in lowercase
- email is stored already canonicalized in lowercase
- no normalized_username column
- no normalized_email column

2.3 Keep security-critical state, drop nonessential metadata
Keep:
- password hash
- account status
- RBAC mappings
- session/device runtime state
- refresh token rotation state

Drop:
- extra browser/platform/debug columns unless currently needed
- audit history tables
- analytics-style columns
- duplicated status booleans when one status field is enough

2.4 Sensitive data protection
- password uses Argon2id hash
- phone number is encrypted before persistence
- TOTP secret is encrypted before persistence
- refresh token is stored as hash only
- backup code is stored as hash only

2.5 Hard delete runtime state
Hard delete only:
- devices
- sessions
- refresh_tokens

No soft delete columns on those tables.

2.6 Keep non-core concerns outside iam schema
Outside this schema:
- one-time token lifecycle
- mail queue
- audit logs
- stdout/stderr logs
- Redis token bucket data
- Redis stream messages

3. Tables Overview

Required tables:
- iam.users
- iam.mfa_totp_credentials
- iam.mfa_backup_codes
- iam.roles
- iam.permissions
- iam.role_permissions
- iam.user_roles
- iam.devices
- iam.sessions
- iam.refresh_tokens

Not included:
- user_password_credentials
- normalized_username
- normalized_email
- email_verified boolean
- refresh_token_families
- audit tables
- one_time_token tables

4. Table Specifications

4.1 iam.users

Purpose
- Stores the main identity record.
- This is the root table of IAM.

Columns
- id : ulid, primary key
- username : text, required, unique
- email : text, required, unique
- full_name : text, required
- phone_number_ciphertext : text, nullable
- password_hash : text, required
- status : text, required
- authz_version : bigint, required, default 1
- locked_at : timestamptz, nullable
- created_at : timestamptz, required
- updated_at : timestamptz, required

Allowed status values
- pending_email_verification
- active
- locked

Constraints
- primary key on id
- unique on username
- unique on email
- check username <> ''
- check email <> ''
- check full_name <> ''
- check status in allowed values

Indexes
- unique index on username
- unique index on email
- index on status

Rules
- username is the login identifier
- username is stored in canonical lowercase form
- email is stored in canonical lowercase form
- no separate email_verified boolean is needed
- email verification is represented by status transition:
  - pending_email_verification -> active
- password_hash stores Argon2id encoded hash
- phone_number_ciphertext is the only stored phone number field
- no plaintext phone number column exists
- authz_version is used for near-real-time RBAC invalidation

Why this table is minimal but sufficient
- no duplicated normalized columns
- no split password table
- no redundant email_verified boolean
- no extra login analytics columns in core row

4.2 iam.mfa_totp_credentials

Purpose
- Stores one current TOTP secret per user.

Columns
- user_id : ulid, primary key, foreign key -> iam.users(id)
- secret_ciphertext : text, required
- enabled : boolean, required, default false
- created_at : timestamptz, required
- updated_at : timestamptz, required

Constraints
- primary key on user_id
- foreign key user_id -> iam.users(id) on delete restrict

Indexes
- index on enabled

Rules
- one row per user at most
- secret is encrypted at rest
- enabled=false means enrolled but not yet active
- enabled=true means MFA is enforceable

Why no extra columns
- no issuer column needed in database core
- no account_name column needed in database core
- no last_used_at unless product explicitly needs it

4.3 iam.mfa_backup_codes

Purpose
- Stores backup codes in hashed form.

Columns
- id : ulid, primary key
- user_id : ulid, required, foreign key -> iam.users(id)
- code_hash : text, required
- used_at : timestamptz, nullable
- created_at : timestamptz, required

Constraints
- primary key on id
- foreign key user_id -> iam.users(id) on delete restrict

Indexes
- index on user_id
- index on user_id, used_at

Rules
- backup codes are never stored in plaintext
- one code can be used once
- regenerating codes may delete or replace all old unused rows by application policy

4.4 iam.roles

Purpose
- Stores RBAC roles.

Columns
- id : ulid, primary key
- code : text, required, unique
- name : text, required
- description : text, nullable
- is_system : boolean, required, default false
- created_at : timestamptz, required
- updated_at : timestamptz, required

Constraints
- primary key on id
- unique on code

Indexes
- unique index on code
- index on is_system

Rules
- seeded role user must exist
- admin may manage roles according to business rules
- system roles may be protected from deletion at application layer

Why keep these columns
- code is stable machine identifier
- name/description support admin UI
- is_system protects seeded roles

4.5 iam.permissions

Purpose
- Stores the immutable permission catalog.

Columns
- id : ulid, primary key
- code : text, required, unique
- description : text, nullable
- created_at : timestamptz, required
- updated_at : timestamptz, required

Constraints
- primary key on id
- unique on code

Indexes
- unique index on code

Rules
- permissions are seeded data
- admin cannot create permissions
- admin cannot delete permissions
- admin may update description only

Why this table is intentionally small
- module/resource/action columns are not required if code naming convention is strong
- permission code itself is the canonical identifier
- example code:
  - iam:user:read
  - iam:user:update
  - iam:role:assign

4.6 iam.role_permissions

Purpose
- Many-to-many mapping between roles and permissions.

Columns
- role_id : ulid, required, foreign key -> iam.roles(id)
- permission_id : ulid, required, foreign key -> iam.permissions(id)

Constraints
- composite primary key on (role_id, permission_id)
- foreign key role_id -> iam.roles(id) on delete cascade
- foreign key permission_id -> iam.permissions(id) on delete restrict

Indexes
- primary key on (role_id, permission_id)
- index on permission_id

Rules
- deleting a role removes only the mapping rows
- deleting permissions is not part of business flow

4.7 iam.user_roles

Purpose
- Many-to-many mapping between users and roles.

Columns
- user_id : ulid, required, foreign key -> iam.users(id)
- role_id : ulid, required, foreign key -> iam.roles(id)
- created_at : timestamptz, required

Constraints
- composite primary key on (user_id, role_id)
- foreign key user_id -> iam.users(id) on delete restrict
- foreign key role_id -> iam.roles(id) on delete restrict

Indexes
- primary key on (user_id, role_id)
- index on role_id

Rules
- registration must insert default role user
- any change here must bump users.authz_version in application layer

Why no granted_by column
- current architecture logs admin actions to stdout/stderr
- not storing grant history inside core IAM keeps schema cleaner

4.8 iam.devices

Purpose
- Stores active or known devices for a user.
- Runtime state only.

Columns
- id : ulid, primary key
- user_id : ulid, required, foreign key -> iam.users(id)
- device_label : text, nullable
- fingerprint_hash : text, required
- proof_key_thumbprint : text, nullable
- trusted : boolean, required, default false
- verified_at : timestamptz, nullable
- last_seen_at : timestamptz, nullable
- created_at : timestamptz, required
- updated_at : timestamptz, required

Constraints
- primary key on id
- foreign key user_id -> iam.users(id) on delete restrict

Indexes
- index on user_id
- index on user_id, last_seen_at desc
- index on trusted
- index on proof_key_thumbprint

Rules
- devices are hard deleted only
- no deleted_at
- no revoked_at
- fingerprint_hash is a risk/binding signal, not sole proof of identity
- proof_key_thumbprint supports stronger binding when available

Why this table is lean enough
- no last_ip
- no raw user_agent
- no platform/browser columns
- no extra fingerprint detail columns
- those belong to logs or risk pipeline, not core runtime DB

4.9 iam.sessions

Purpose
- Stores active login sessions.
- Runtime state only.

Columns
- id : ulid, primary key
- user_id : ulid, required, foreign key -> iam.users(id)
- device_id : ulid, required, foreign key -> iam.devices(id)
- auth_method : text, required
- mfa_verified : boolean, required, default false
- proof_key_thumbprint : text, nullable
- expires_at : timestamptz, required
- last_seen_at : timestamptz, nullable
- created_at : timestamptz, required

Allowed auth_method values
- password
- password_mfa
- recovery_code

Constraints
- primary key on id
- foreign key user_id -> iam.users(id) on delete restrict
- foreign key device_id -> iam.devices(id) on delete cascade
- check auth_method in allowed values

Indexes
- index on user_id
- index on device_id
- index on expires_at
- index on user_id, last_seen_at desc

Rules
- sessions are hard deleted only
- logout physically deletes row
- deleting a device cascades to deleting its sessions

Why no updated_at here
- for runtime session, last_seen_at already serves the update signal
- this removes one unnecessary mutable column

4.10 iam.refresh_tokens

Purpose
- Stores refresh token rotation chain per session.
- Runtime state only.

Columns
- id : ulid, primary key
- session_id : ulid, required, foreign key -> iam.sessions(id)
- parent_token_id : ulid, nullable, foreign key -> iam.refresh_tokens(id)
- token_hash : text, required, unique
- status : text, required
- expires_at : timestamptz, required
- used_at : timestamptz, nullable
- created_at : timestamptz, required

Allowed status values
- active
- rotated

Constraints
- primary key on id
- unique on token_hash
- foreign key session_id -> iam.sessions(id) on delete cascade
- foreign key parent_token_id -> iam.refresh_tokens(id) on delete set null
- check status in allowed values

Indexes
- unique index on token_hash
- index on session_id
- index on session_id, status
- index on expires_at

Rules
- refresh token plaintext is never stored
- only hash is stored
- one session has one refresh rotation chain
- no separate refresh_token_families table is required
- rotation:
  - old active token -> rotated
  - new token row -> active

Why this is cleaner
- family table is unnecessary because session_id already defines the chain scope
- one fewer table
- one fewer join
- simpler code path for refresh logic

5. Relationship Summary

users
- 1 -> 0..1 mfa_totp_credentials
- 1 -> many mfa_backup_codes
- 1 -> many user_roles
- 1 -> many devices
- 1 -> many sessions

roles
- many -> many permissions through role_permissions
- many -> many users through user_roles

devices
- 1 -> many sessions

sessions
- 1 -> many refresh_tokens

6. Hard Delete Policy

Hard delete only:
- iam.devices
- iam.sessions
- iam.refresh_tokens

No soft delete columns are allowed on these runtime tables.

Restrict delete:
- iam.users
- iam.permissions

Application-controlled delete:
- iam.roles, subject to business rules

7. Registration Flow Touchpoints

Register account writes:
- iam.users
- iam.user_roles

Register account does not write:
- iam.devices
- iam.sessions
- iam.refresh_tokens

Register account does not store:
- verification token
- mail job
- rate limit counters

Register account logic:
- create user
- store Argon2id password_hash in iam.users
- store encrypted phone_number_ciphertext when provided
- assign default role user in iam.user_roles
- call one-time-token service outside DB
- publish mail job to Redis Stream outside DB

8. Login Flow Touchpoints

Successful login reads:
- iam.users
- iam.mfa_totp_credentials
- iam.mfa_backup_codes
- iam.user_roles
- iam.role_permissions
- iam.permissions

Successful login writes:
- iam.devices
- iam.sessions
- iam.refresh_tokens

Optional user update:
- users.updated_at only when business logic explicitly updates user state
- no need to mutate users row on every login unless product needs it

9. Verify Email Flow Touchpoints

Verify email updates:
- iam.users.status
- iam.users.updated_at

State transition:
- pending_email_verification -> active

10. Excluded by Design

No user_password_credentials table
Reason:
- current password is simple and sufficient inside iam.users

No normalized_username
Reason:
- username is stored already canonicalized

No normalized_email
Reason:
- email is stored already canonicalized

No email_verified boolean
Reason:
- status already expresses verification state

No refresh_token_families table
Reason:
- session_id is enough to define token rotation scope

No audit tables
Reason:
- logs go to stdout/stderr and external observability pipeline

No one-time-token tables
Reason:
- token lifecycle belongs to separate service

11. Recommended Seed Data

roles
- user
- admin

permissions
- seed all required IAM permission codes

role_permissions
- map baseline permissions to user
- map full administrative permissions to admin

12. Final Schema Summary

Required tables:
- iam.users
- iam.mfa_totp_credentials
- iam.mfa_backup_codes
- iam.roles
- iam.permissions
- iam.role_permissions
- iam.user_roles
- iam.devices
- iam.sessions
- iam.refresh_tokens

This design is production-ready because it keeps:
- strict uniqueness
- strict referential integrity
- minimal but sufficient runtime state
- strong protection for sensitive fields
- low schema complexity
- low join complexity
- low cognitive load in service code