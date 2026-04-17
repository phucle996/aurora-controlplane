AUTH REGISTER SPEC — AI OPTIMIZED

Flow metadata
- Flow name: Register account
- Flow code: AUTH-REG-001
- Module: auth
- Business group: Authentication
- Actor: Guest
- Endpoint: POST /api/v1/auth/register
- Authentication required: No
- Source: :contentReference[oaicite:0]{index=0}

1. Purpose
Create a new control plane account using:
- full_name
- email
- username
- phone_number (optional)
- password
- re_password

This flow must:
- create user
- create password credential
- assign default role = user
- request email verification token from One Time Token service
- build verification link
- publish verification email job to Redis Stream

This flow must not:
- create session
- create device
- issue access token
- issue refresh token
- auto-login user

2. Login rule
- Login method after registration: username + password
- Email is not used for login
- Phone number is not used for login
- No phone verification flow

3. Request contract
Headers:
- Content-Type: application/json
- Optional:
  - X-Request-ID
  - X-Forwarded-For
  - User-Agent

Request body example:
{
  "full_name": "Nguyen Van A",
  "email": "admin@example.com",
  "username": "aurora_admin",
  "phone_number": "+84901234567",
  "password": "StrongPassword123!",
  "re_password": "StrongPassword123!"
}

4. Field rules
full_name
- required
- string
- trim leading/trailing spaces
- length 2..120
- allow unicode letters
- stored as display/profile full name

email
- required
- string
- trim
- normalize lowercase
- must be valid email format
- must be unique across users

username
- required
- string
- trim
- normalize lowercase
- length 3..32
- allowed chars: a-z, 0-9, underscore, dot, hyphen
- reserved usernames forbidden
- must be unique across users

phone_number
- optional
- nullable or empty string allowed
- trim
- normalize before encryption
- stored only as encrypted profile/contact data
- not used for login
- not used for verification
- uniqueness not required

password
- required
- string
- must satisfy password policy
- must be hashed with Argon2id
- must never be logged
- must never be stored in plain text

re_password
- required
- string
- must exactly match password
- validation only
- must never be stored
- must never be logged

5. Security rules
Password storage
- algorithm: Argon2id
- store only password hash
- never encrypt password for storage
- never store plain password
- hash metadata may be encoded in hash string or stored with hash
- final Argon2id parameters are centrally configurable by security policy

Recommended Argon2id controls
- memory: policy-defined
- iterations: policy-defined
- parallelism: policy-defined
- salt: cryptographically secure random salt per password
- output length: policy-defined

Phone number storage
- phone number must be encrypted before persistence
- store ciphertext and required encryption metadata only
- if phone number is empty, store null
- plain phone number must never be written to persistent storage
- decryption allowed only in authorized read flows
- encryption keys must never appear in request payloads or logs

Sensitive data never logged
- password
- re_password
- plain verification token
- plain phone number
- full cryptographic material

6. Business rules
- email verification is required after successful registration
- registration does not auto-login
- username must be unique
- email must be unique
- phone number may be empty
- successful registration must assign exactly one default role: user
- email verification token generation is delegated to One Time Token service
- auth module does not generate verification token locally
- when a new verification token is issued for the same purpose, previous active token must be invalidated by OTT service policy

7. Preconditions
Flow may proceed only if:
- caller is guest
- rate limit allows request
- payload passes schema validation
- password == re_password
- username does not already exist
- email does not already exist
- seeded role user exists
- self-registration is enabled if platform policy requires it

8. Main flow
Step 1 — Receive request
- parse JSON body
- collect request metadata:
  - request_id
  - ip
  - user_agent

Step 2 — Apply rate limit
Apply token bucket at least by:
- IP
- username
- email

Redis keys:
- rl:auth:register:ip:{ip}
- rl:auth:register:username:{normalized_username}
- rl:auth:register:email:{normalized_email}

Step 3 — Normalize input
- trim full_name
- trim + lowercase email
- trim + lowercase username
- normalize phone_number if present

Step 4 — Validate input
Reject if:
- required fields missing
- full_name invalid
- email invalid
- username invalid
- username reserved
- phone_number invalid when present
- password policy fails
- password != re_password

Step 5 — Start database transaction

Step 6 — Check uniqueness
- check normalized_username
- check normalized_email
- database unique constraints remain final source of truth

Step 7 — Encrypt phone number
If phone_number is present and non-empty:
- encrypt normalized phone number
- produce ciphertext + encryption metadata

If encryption fails:
- rollback if transaction started
- return 500 PHONE_ENCRYPTION_FAILED

Step 8 — Create user
Insert user with:
- full_name
- email
- username
- phone_number_ciphertext
- encryption metadata if used
- status = pending_email_verification

Step 9 — Hash password
- hash password using Argon2id
- never persist plain password or re_password

If hashing fails:
- rollback transaction
- return 500 PASSWORD_HASH_FAILED

Step 10 — Create password credential
Insert password credential linked to user

Step 11 — Assign default role
- resolve seeded role user
- create user-role mapping

If role assignment fails:
- rollback transaction
- return 500 DEFAULT_ROLE_ASSIGNMENT_FAILED

Step 12 — Commit transaction
Transaction must commit:
- user
- encrypted phone data
- password credential
- default role assignment

Step 13 — Request email verification token
After commit:
- call One Time Token service
- purpose: verify_email
- subject_type: user
- subject_id: user_id
- channel: email
- destination: normalized email

OTT service is responsible for:
- issuing new token
- invalidating previously active token for same purpose per policy

Expected response:
- token
- ttl or expires_at

Step 14 — Build verification link
- auth module builds verification URL from returned token

Step 15 — Publish mail job
Publish to Redis Stream:
- stream: stream:mail:outgoing

Payload should include at least:
- type = verify_email
- user_id
- email
- full_name
- template_key
- verification_link
- request_id
- created_at

Auth module does not send email directly

Step 16 — Emit logs
- success logs -> stdout
- failure logs -> stderr

9. Transaction boundary
Database transaction covers only:
- create user
- create password credential
- assign default role

Outside transaction:
- OTT service call
- Redis Stream mail publish

Rule:
- phone encryption must complete before user row is written

10. Database touchpoints
users
- id
- full_name
- username
- email
- phone_number_ciphertext
- encryption metadata if required
- status
- created_at
- updated_at

user_passwords
- user_id
- password_hash
- hash_algorithm = argon2id
- created_at
- updated_at

user_roles
- user_id
- role_id
- created_at

roles
- seeded role catalog must already contain:
  - user

Note
- no local email verification token table is required if token lifecycle is fully owned by OTT service

11. Service touchpoints
One Time Token service
Responsibilities:
- generate email verification token
- enforce single active token policy for same purpose
- return token metadata to auth module

Phone encryption service
Responsibilities:
- encrypt normalized phone number before persistence
- return ciphertext and encryption metadata

12. Success response
HTTP 201 Created
{
  "code": "REGISTERED",
  "message": "Account registered successfully. Please verify your email."
}

13. Error mapping
400 Bad Request
- VALIDATION_ERROR
- PASSWORD_CONFIRMATION_MISMATCH
- INVALID_USERNAME
- INVALID_EMAIL
- INVALID_PHONE_NUMBER
- WEAK_PASSWORD

403 Forbidden
- REGISTRATION_DISABLED
- EMAIL_DOMAIN_BLOCKED

409 Conflict
- USERNAME_ALREADY_EXISTS
- EMAIL_ALREADY_EXISTS

413 Payload Too Large
- REQUEST_ENTITY_TOO_LARGE

429 Too Many Requests
- RATE_LIMITED

500 Internal Server Error
- INTERNAL_ERROR
- DEFAULT_ROLE_ASSIGNMENT_FAILED
- PHONE_ENCRYPTION_FAILED
- PASSWORD_HASH_FAILED
- OTT_GENERATION_FAILED
- MAIL_JOB_PUBLISH_FAILED

14. Failure semantics
Validation or input failure
- reject before database work

Uniqueness conflict
- rollback transaction if started
- return:
  - 409 USERNAME_ALREADY_EXISTS
  - or 409 EMAIL_ALREADY_EXISTS

Concurrent race
- database unique constraints are final source of truth
- map unique violation to correct 409 error

Password hash failure
- rollback transaction
- return 500 PASSWORD_HASH_FAILED

Phone encryption failure
- rollback if transaction started
- return 500 PHONE_ENCRYPTION_FAILED

Default role missing
- rollback transaction
- return 500 DEFAULT_ROLE_ASSIGNMENT_FAILED

OTT failure after commit
State remains committed:
- user exists
- password exists
- default role exists
- account remains pending_email_verification

Return:
- 500 OTT_GENERATION_FAILED

Mail publish failure after OTT success
State remains committed:
- user exists
- password exists
- default role exists
- token may already exist in OTT service
- account remains pending_email_verification

Return:
- 500 MAIL_JOB_PUBLISH_FAILED

Server succeeded but client missed response
- account may already exist
- retry may hit conflict
- logs must support correlation by request_id

15. Logging rules
stdout
- register success
- default role assigned
- verification token requested successfully
- mail job published successfully

stderr
- validation failure
- uniqueness conflict
- phone encryption failure
- password hash failure
- database failure
- default role assignment failure
- OTT failure
- Redis publish failure

Never log
- password
- re_password
- plain verification token
- plain phone number
- cryptographic secrets

Rule
- logging failure itself must not break business flow

16. Idempotency
- this flow is not idempotent
- retrying same successful payload will usually return:
  - 409 USERNAME_ALREADY_EXISTS
  - or 409 EMAIL_ALREADY_EXISTS

17. Final canonical rules
- register input = full_name + email + username + phone_number optional + password + re_password
- login method = username + password
- phone number is optional profile data only
- phone number must be encrypted at rest
- password must be hashed with Argon2id
- re_password is validation-only
- success must assign default role user
- auth module must call OTT service for email verification token
- auth module builds verification link from returned token
- verification email is dispatched asynchronously via Redis Stream
- register flow must not create session
- register flow must not create device
- register flow must not issue access token
- register flow must not issue refresh token