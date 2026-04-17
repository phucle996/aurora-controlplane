IAM FLOW SPEC — CONTROL PLANE

Scope
- This IAM belongs to the control plane only.
- No service-to-service or internal auth API is exposed in this spec.
- Login method is username + password only.
- Access token: JWT, 10 minutes.
- Refresh token: opaque token, 5 days, rotation required.
- Session and device are runtime state.
- Session and device use hard delete only. No soft delete.
- Permission catalog is seeded into database.
- Permissions cannot be created or deleted, including by admin.
- Admin may only view permissions and update permission descriptions.
- Rate limiting uses token bucket.
- Audit and operational logs are written to stdout and stderr.
- No separate audit API is exposed.

Route Convention

Public / Client-facing
- /api/v1/auth/*
- /api/v1/account/*
- /api/v1/me/*
- /api/v1/sessions/*
- /api/v1/devices/*
- /api/v1/security/*

Admin / Backoffice
- /admin/v1/users/*
- /admin/v1/roles/*
- /admin/v1/permissions/*
- /admin/v1/sessions/*
- /admin/v1/devices/*
- /admin/v1/security/*


1. Business Group: Authentication

Module: Public Auth
Actor: Guest / User

- Register account
  Endpoint: POST /api/v1/auth/register
  Actor: Guest

- Verify email
  Endpoint: POST /api/v1/auth/verify-email
  Actor: Guest
  Note: when a new verification token is issued, the previous token is invalidated. There is no separate resend flow.

- Login with username + password
  Endpoint: POST /api/v1/auth/login
  Actor: Guest

- Start MFA challenge
  Endpoint: POST /api/v1/auth/mfa/challenge
  Actor: Guest / User

- Verify MFA code
  Endpoint: POST /api/v1/auth/mfa/verify
  Actor: Guest / User

- Login with backup code
  Endpoint: POST /api/v1/auth/mfa/backup-code
  Actor: Guest / User

- Refresh access token
  Endpoint: POST /api/v1/auth/refresh
  Actor: User

- Logout current session
  Endpoint: DELETE /api/v1/auth/logout
  Actor: User

- Logout all sessions
  Endpoint: DELETE /api/v1/auth/logout-all
  Actor: User

- Re-authenticate before sensitive action
  Endpoint: POST /api/v1/auth/re-auth
  Actor: User


Module: Auth Recovery
Actor: Guest / User

- Forgot password
  Endpoint: POST /api/v1/auth/forgot-password
  Actor: Guest

- Reset password
  Endpoint: POST /api/v1/auth/reset-password
  Actor: Guest

- Change password
  Endpoint: PUT /api/v1/account/password
  Actor: User

- Recover account when MFA is lost
  Endpoint Group: /api/v1/auth/recovery/*
  Actor: Guest / User


2. Business Group: Session Management

Module: User Session
Actor: User

- View current session
  Endpoint: GET /api/v1/sessions/current
  Actor: User

- List my sessions
  Endpoint: GET /api/v1/sessions
  Actor: User

- Delete one session
  Endpoint: DELETE /api/v1/sessions/:session_id
  Actor: User

- Delete all other sessions
  Endpoint: DELETE /api/v1/sessions/others
  Actor: User

- Extend trusted session
  Endpoint: POST /api/v1/sessions/:session_id/extend
  Actor: User


Module: Admin Session
Actor: Admin

- List user sessions
  Endpoint: GET /admin/v1/users/:user_id/sessions
  Actor: Admin

- View session detail
  Endpoint: GET /admin/v1/sessions/:session_id
  Actor: Admin

- Delete one session
  Endpoint: DELETE /admin/v1/sessions/:session_id
  Actor: Admin

- Delete all sessions of a user
  Endpoint: DELETE /admin/v1/users/:user_id/sessions
  Actor: Admin

- Force re-authentication for user
  Endpoint: POST /admin/v1/users/:user_id/force-re-auth
  Actor: Admin

Session Runtime Rule
- Sessions are runtime state only.
- Logout, revoke, expiration cleanup, and admin removal physically delete session records.
- No deleted_at column.
- No soft delete behavior.


3. Business Group: Device Management

Module: User Device
Actor: User

- Register device after login
  Endpoint: POST /api/v1/devices/register
  Actor: User

- Bind device key / proof key
  Endpoint: POST /api/v1/devices/:device_id/bind-key
  Actor: User

- List my devices
  Endpoint: GET /api/v1/devices
  Actor: User

- View my device detail
  Endpoint: GET /api/v1/devices/:device_id
  Actor: User

- Rename device
  Endpoint: PUT /api/v1/devices/:device_id
  Actor: User

- Trust device
  Endpoint: POST /api/v1/devices/:device_id/trust
  Actor: User

- Untrust device
  Endpoint: POST /api/v1/devices/:device_id/untrust
  Actor: User

- Delete device
  Endpoint: DELETE /api/v1/devices/:device_id
  Actor: User

- Verify new device
  Endpoint: POST /api/v1/devices/:device_id/verify
  Actor: User

- View device activity
  Endpoint: GET /api/v1/devices/:device_id/activity
  Actor: User


Module: Admin Device
Actor: Admin

- List devices of a user
  Endpoint: GET /admin/v1/users/:user_id/devices
  Actor: Admin

- View device detail
  Endpoint: GET /admin/v1/devices/:device_id
  Actor: Admin

- Delete device
  Endpoint: DELETE /admin/v1/devices/:device_id
  Actor: Admin

- Mark device high risk
  Endpoint: POST /admin/v1/devices/:device_id/flag
  Actor: Admin

- Remove high-risk flag
  Endpoint: POST /admin/v1/devices/:device_id/unflag
  Actor: Admin

Device Runtime Rule
- Devices are runtime state only.
- User delete and admin delete physically remove device records.
- No deleted_at column.
- No soft delete behavior.
- Historical investigation does not rely on deleted device records.


4. Business Group: Account / Profile

Module: Account Self-Service
Actor: User

- Get my profile
  Endpoint: GET /api/v1/me/profile
  Actor: User

- Update my profile
  Endpoint: PUT /api/v1/me/profile
  Actor: User

- Get my security settings
  Endpoint: GET /api/v1/me/security
  Actor: User

- Update email
  Endpoint: PUT /api/v1/account/email
  Actor: User


Module: Account Admin
Actor: Admin

- View user profile
  Endpoint: GET /admin/v1/users/:user_id
  Actor: Admin

- Lock user
  Endpoint: POST /admin/v1/users/:user_id/lock
  Actor: Admin

- Unlock user
  Endpoint: POST /admin/v1/users/:user_id/unlock
  Actor: Admin

- Reset password for user
  Endpoint: POST /admin/v1/users/:user_id/reset-password
  Actor: Admin

- Reset MFA for user
  Endpoint: POST /admin/v1/users/:user_id/reset-mfa
  Actor: Admin


5. Business Group: MFA

Module: MFA User
Actor: User

- Enroll TOTP
  Endpoint: POST /api/v1/account/mfa/totp/enroll
  Actor: User

- Confirm TOTP setup
  Endpoint: POST /api/v1/account/mfa/totp/confirm
  Actor: User

- Disable TOTP
  Endpoint: POST /api/v1/account/mfa/totp/disable
  Actor: User

- Generate backup codes
  Endpoint: POST /api/v1/account/mfa/backup-codes
  Actor: User

- Regenerate backup codes
  Endpoint: POST /api/v1/account/mfa/backup-codes/regenerate
  Actor: User

- List MFA methods
  Endpoint: GET /api/v1/account/mfa/methods
  Actor: User

- Set default MFA method
  Endpoint: PUT /api/v1/account/mfa/default-method
  Actor: User


Module: MFA Admin
Actor: Admin

- View MFA methods of user
  Endpoint: GET /admin/v1/users/:user_id/mfa
  Actor: Admin

- Reset MFA
  Endpoint: POST /admin/v1/users/:user_id/mfa/reset
  Actor: Admin

- Disable MFA
  Endpoint: POST /admin/v1/users/:user_id/mfa/disable
  Actor: Admin


6. Business Group: Authorization / RBAC

Module: Permission Catalog
Actor: Admin

Permission Rule
- Permissions are seeded data.
- Permissions cannot be created.
- Permissions cannot be deleted.
- Admin may only read permissions and update descriptions.

Flows

- List permissions
  Endpoint: GET /admin/v1/permissions
  Actor: Admin

- View permission detail
  Endpoint: GET /admin/v1/permissions/:permission_id
  Actor: Admin

- Update permission description
  Endpoint: PUT /admin/v1/permissions/:permission_id/description
  Actor: Admin


Module: Role Management
Actor: Admin

- List roles
  Endpoint: GET /admin/v1/roles
  Actor: Admin

- Create role
  Endpoint: POST /admin/v1/roles
  Actor: Admin

- View role detail
  Endpoint: GET /admin/v1/roles/:role_id
  Actor: Admin

- Update role
  Endpoint: PUT /admin/v1/roles/:role_id
  Actor: Admin

- Delete role
  Endpoint: DELETE /admin/v1/roles/:role_id
  Actor: Admin

- Clone role
  Endpoint: POST /admin/v1/roles/:role_id/clone
  Actor: Admin

- Attach permissions to role
  Endpoint: POST /admin/v1/roles/:role_id/permissions
  Actor: Admin

- Remove permissions from role
  Endpoint: DELETE /admin/v1/roles/:role_id/permissions
  Actor: Admin

- List permissions of role
  Endpoint: GET /admin/v1/roles/:role_id/permissions
  Actor: Admin


Module: User Role Assignment
Actor: Admin

- List roles of user
  Endpoint: GET /admin/v1/users/:user_id/roles
  Actor: Admin

- Assign roles to user
  Endpoint: POST /admin/v1/users/:user_id/roles
  Actor: Admin

- Remove roles from user
  Endpoint: DELETE /admin/v1/users/:user_id/roles
  Actor: Admin

- Get effective permissions of user
  Endpoint: GET /admin/v1/users/:user_id/effective-permissions
  Actor: Admin

- Bump authz version of user
  Endpoint: POST /admin/v1/users/:user_id/authz-version/bump
  Actor: Admin


7. Business Group: Security / Risk

Module: Security User
Actor: User

- View my security events
  Endpoint: GET /api/v1/security/events
  Actor: User

- View suspicious login history
  Endpoint: GET /api/v1/security/logins
  Actor: User

- Confirm this activity was me
  Endpoint: POST /api/v1/security/events/:event_id/confirm
  Actor: User

- Report this activity was not me
  Endpoint: POST /api/v1/security/events/:event_id/report
  Actor: User


Module: Security Admin
Actor: Admin

- List security events
  Endpoint: GET /admin/v1/security/events
  Actor: Admin

- View security event detail
  Endpoint: GET /admin/v1/security/events/:event_id
  Actor: Admin

- Mark security event resolved
  Endpoint: POST /admin/v1/security/events/:event_id/resolve
  Actor: Admin

- Force secure account
  Endpoint: POST /admin/v1/users/:user_id/security/secure
  Actor: Admin

- Force delete all sessions of user
  Endpoint: DELETE /admin/v1/users/:user_id/security/sessions
  Actor: Admin


Module: Rate Limit Policy
Actor: System / Platform

Policy Rule
- Token bucket is used for all auth-sensitive and security-sensitive endpoints.
- Rate limits may be applied by IP, username, session scope, device scope, and endpoint scope depending on the flow.

Flows

- Rate limit login
  Endpoint: /api/v1/auth/login
  Actor: System

- Rate limit refresh
  Endpoint: /api/v1/auth/refresh
  Actor: System

- Rate limit forgot password
  Endpoint: /api/v1/auth/forgot-password
  Actor: System

- Rate limit reset password
  Endpoint: /api/v1/auth/reset-password
  Actor: System

- Rate limit MFA verify
  Endpoint: /api/v1/auth/mfa/verify
  Actor: System

- Rate limit device verification
  Endpoint: /api/v1/devices/:device_id/verify
  Actor: System


8. Business Group: Audit / Logging

Module: Operational Logging
Actor: System / Platform

Logging Rule
- No separate audit API is exposed.
- Business logs and security logs are emitted to stdout and stderr.
- Collection is handled by runtime, container platform, or observability stack.

Flows / Outputs

- Auth success log
  Output: stdout
  Actor: System

- Auth failure log
  Output: stderr
  Actor: System

- MFA event log
  Output: stdout / stderr
  Actor: System

- Password reset log
  Output: stdout / stderr
  Actor: System

- Session deletion log
  Output: stdout
  Actor: System

- Device deletion log
  Output: stdout
  Actor: System

- Role assignment log
  Output: stdout
  Actor: System

- Security event log
  Output: stdout / stderr
  Actor: System

- Admin action log
  Output: stdout / stderr
  Actor: System


9. Actor Mapping

Guest
- register account
- verify email
- login with username + password
- forgot password
- reset password
- complete MFA challenge during login

User
- refresh token
- logout current session
- logout all sessions
- manage own sessions
- manage own devices
- manage own MFA
- update own profile
- update own email
- change own password
- view own security events

Admin
- manage users
- manage sessions
- manage devices
- manage roles
- view permission catalog
- update permission descriptions
- review security events
- reset password
- reset MFA
- lock and unlock users
- force secure accounts

System
- apply token bucket rate limits
- emit stdout/stderr logs
- perform security and runtime enforcement


10. Code Module Mapping

auth
- register
- verify-email
- login
- refresh
- logout
- logout-all
- re-auth
- recovery

account
- profile
- email
- password
- security-settings

mfa
- totp
- backup-codes
- default-method
- admin-reset

session
- current
- list
- delete-one
- delete-others
- extend-trusted

device
- register
- bind-key
- list
- detail
- rename
- trust
- untrust
- delete
- verify
- activity

rbac
- permission-catalog
- permission-description
- roles
- role-permissions
- user-roles
- effective-permissions
- authz-version

security
- events
- suspicious-logins
- secure-account
- rate-limit-policy

logging
- stdout-business-log
- stderr-error-log
- stderr-security-log


11. Final Route Set

Public
- /api/v1/auth/*
- /api/v1/account/*
- /api/v1/me/*
- /api/v1/sessions/*
- /api/v1/devices/*
- /api/v1/security/*

Admin
- /admin/v1/users/*
- /admin/v1/roles/*
- /admin/v1/permissions/*
- /admin/v1/sessions/*
- /admin/v1/devices/*
- /admin/v1/security/*


12. Hard Rules Summary

- Control plane IAM only.
- No service-to-service API in this spec.
- Username login only.
- No passkey / WebAuthn login.
- No phone verification flow.
- No resend verification flow.
- Verify email is the only email verification flow.
- Issuing a new email verification token invalidates the previous token.
- Permissions are seeded and immutable in identity, except description updates.
- Rate limiting uses token bucket.
- Sessions are hard deleted only.
- Devices are hard deleted only.
- Historical visibility comes from security events and logs, not from deleted runtime records.
- Audit and operational logging go to stdout and stderr only.
