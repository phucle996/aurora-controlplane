# SMTP Module Clean-Room Load Test Plan

## Goal

- Run full SMTP API validation on a clean DB + clean Redis.
- Measure behavior by phase: `load`, `stress`, `spike`, `soak`.
- Compare with previous mixed IAM+SMTP baseline in:
  - `internal/iam/docs/loadtest/loadtest-results.md`

## Scope

All SMTP routes from `internal/smtp/route.go`:

- Aggregation:
  - `GET /api/v1/smtp/aggregation`
- Consumers:
  - `GET /api/v1/smtp/consumers`
  - `GET /api/v1/smtp/consumers/:id`
  - `POST /api/v1/smtp/consumers/try-connect`
  - `POST /api/v1/smtp/consumers`
  - `PUT /api/v1/smtp/consumers/:id`
  - `DELETE /api/v1/smtp/consumers/:id`
  - `GET /api/v1/smtp/consumers/options`
- Templates:
  - `GET /api/v1/smtp/templates`
  - `GET /api/v1/smtp/templates/:id`
  - `POST /api/v1/smtp/templates`
  - `PUT /api/v1/smtp/templates/:id`
  - `DELETE /api/v1/smtp/templates/:id`
- Gateways:
  - `GET /api/v1/smtp/gateways`
  - `GET /api/v1/smtp/gateways/:id`
  - `GET /api/v1/smtp/gateways/:id/detail`
  - `PUT /api/v1/smtp/gateways/:id/templates`
  - `PUT /api/v1/smtp/gateways/:id/endpoints`
  - `POST /api/v1/smtp/gateways/:id/start`
  - `POST /api/v1/smtp/gateways/:id/drain`
  - `POST /api/v1/smtp/gateways/:id/disable`
  - `POST /api/v1/smtp/gateways`
  - `PUT /api/v1/smtp/gateways/:id`
  - `DELETE /api/v1/smtp/gateways/:id`
- Endpoints:
  - `GET /api/v1/smtp/endpoints`
  - `GET /api/v1/smtp/endpoints/:id`
  - `POST /api/v1/smtp/endpoints/try-connect`
  - `POST /api/v1/smtp/endpoints`
  - `PUT /api/v1/smtp/endpoints/:id`
  - `DELETE /api/v1/smtp/endpoints/:id`
- Runtime:
  - `GET /api/v1/smtp/runtime/activity-logs`
  - `GET /api/v1/smtp/runtime/delivery-attempts`
  - `GET /api/v1/smtp/runtime/heartbeats`
  - `GET /api/v1/smtp/runtime/gateway-assignments`
  - `GET /api/v1/smtp/runtime/consumer-assignments`
  - `POST /api/v1/smtp/runtime/reconcile`

## Auth + Middleware Constraints

- SMTP routes are guarded by:
  - `Access()`
  - `RequireDeviceID()`
  - `RequirePermission(...)`
- SMTP routes currently have no `RateLimit()` middleware.
- Test harness must send cookie set:
  - `access_token`
  - `refresh_token`
  - `device_id`
  - `refresh_token_hash`
  - `workspace_id`

## Clean-Room Procedure

1. Stop app services/processes.
2. Drop and recreate Postgres database `aurora`.
3. Flush Redis (`FLUSHALL`) to clear tokens, role cache, and runtime cache.
4. Start app and wait for readiness.
5. Seed load test user + SMTP permissions.
6. Create or resolve one active workspace and one zone for fixtures.

## Multi Replica Strategy

- Replica A: systemd service (`aurora-controlplane.service`) on `:8080`.
- Replica B: second process with same DB/Redis on `:8081`.
- Run same k6 suite against each replica while both are up.
- Verify state coherence by running CRUD on one replica and read/list on the other.

## Test Phases

- `smoke`: one-time full endpoint coverage, strict status expectations.
- `load`: baseline normal traffic (`100 VUs`) on read-heavy + controlled writes.
- `stress`: ramp users gradually until latency/errors degrade.
- `spike`: sudden traffic jump and observe recovery.
- `soak`: long run to detect memory drift/degradation.

## k6 Command Matrix

```bash
# smoke full api
PHASE=smoke k6 run internal/smtp/test/test-load/k6-smtp.js

# load
PHASE=load LOAD_VUS=100 LOAD_DURATION=5m k6 run internal/smtp/test/test-load/k6-smtp.js

# stress
PHASE=stress STRESS_MAX_VUS=400 k6 run internal/smtp/test/test-load/k6-smtp.js

# spike
PHASE=spike SPIKE_VUS=800 k6 run internal/smtp/test/test-load/k6-smtp.js

# soak
PHASE=soak SOAK_VUS=50 SOAK_DURATION=30m k6 run internal/smtp/test/test-load/k6-smtp.js
```

## Expected Artifacts

- `internal/smtp/test/test-load/k6-smtp.js`: executable harness.
- `internal/smtp/test/test-load/result.md`: observed metrics, bottlenecks, failure points.
- Result comparison against previous mixed IAM+SMTP run:
  - p95/p99 latency
  - error rate
  - throughput
  - memory trend
