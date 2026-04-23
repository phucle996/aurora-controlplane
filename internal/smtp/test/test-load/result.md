# SMTP Clean-Room Load Test Result

Date: 2026-04-22  
Module: `internal/smtp`  
Script: `internal/smtp/test/test-load/k6-smtp.js`

## 1) Clean-room + deploy actions executed

- Stopped app service and old manual replica.
- Dropped and recreated Postgres DB `aurora`:
  - `DROP DATABASE IF EXISTS aurora;`
  - `CREATE DATABASE aurora OWNER postgres;`
- Flushed Redis:
  - `redis-cli FLUSHALL`
- Rebuilt and reinstalled app using:
  - `bash install.sh -e /tmp/aurora-install.env`
- Started multi replica:
  - Replica A: systemd `aurora-controlplane.service` on `:8080`
  - Replica B: manual process on `:8081` (`APP_HTTP_PORT=8081`, `GRPC_SERVER_PORT=9091`)
- Seeded test principal:
  - `smtp.loadtest` (status `active`)
  - role `smtp-loadtest-role`
  - 12 SMTP/workspace permissions

## 2) Handler checks (task 1)

- `template_handler.go` and `runtime_handler.go` are in path-specific error mapping style:
  - workspace cookie precondition validated first
  - service/repository path errors mapped inline in each handler
  - added inline comments for flow clarity (transport precondition, path-specific mapping)
- Transport tests still pass:
  - `go test ./internal/smtp/...`

## 3) k6 execution matrix

Ran full matrix on both replicas:

- `smoke`
- `load` (`100 VUs`, `2m`)
- `stress` (`10 -> 80 -> 160 -> 240`, each `30s`, then ramp-down)
- `spike` (`400 VUs`, `15s up`, `45s hold`, `15s down`)
- `soak` (`50 VUs`, `5m`)

## 4) Results (new run)

> Note: k6 `--summary-export` in this setup does not include `p99`; reported `p95` + `max`.

| Phase | Replica | Requests | Req/s | Error % (`http_req_failed`) | p95 (ms) | Max (ms) | `smtp_route_failure` |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| smoke | 8080 | 100 | 234.83 | 3.00 | 5.46 | 209.24 | 0 |
| smoke | 8081 | 98 | 435.99 | 3.06 | 3.88 | 62.27 | 0 |
| load | 8080 | 68,251 | 563.11 | 3.45 | 6.02 | 828.00 | 0 |
| load | 8081 | 68,411 | 565.10 | 3.52 | 4.73 | 243.55 | 0 |
| stress | 8080 | 103,835 | 689.87 | 3.50 | 6.92 | 618.59 | 0 |
| stress | 8081 | 104,004 | 690.98 | 3.51 | 4.95 | 363.63 | 0 |
| spike | 8080 | 81,605 | 1075.42 | 3.53 | 327.37 | 5002.45 | 120 |
| spike | 8081 | 118,635 | 1568.42 | 3.45 | 151.03 | 785.40 | 0 |
| soak | 8080 | 86,071 | 285.95 | 3.37 | 4.34 | 490.70 | 0 |
| soak | 8081 | 86,330 | 286.83 | 3.37 | 3.78 | 267.66 | 0 |

## 5) Soak memory/CPU samples

Sampling interval: 30s, 12 samples per run.

| Soak target | Loaded process | `MemoryCurrent` (service) | RSS service (KB) | CPU service avg (%) | RSS replica2 (KB) | CPU replica2 avg (%) |
| --- | --- | --- | --- | ---: | --- | ---: |
| `BASE_URL=:8080` | systemd replica (`:8080`) | 37,605,376 -> 42,360,832 | 57,336 -> 59,716 | 33.72 | 32,708 -> 32,792 | 0.00 |
| `BASE_URL=:8081` | manual replica (`:8081`) | 26,173,440 -> 26,177,536 | 46,252 -> 46,252 | 17.42 | 56,100 -> 58,140 | 16.86 |

Raw samples:

- `/tmp/smtp2-soak-samples-8080.csv`
- `/tmp/smtp2-soak-samples-8081.csv`

## 6) Comparison with previous test

### 6.1 Compare against previous SMTP-only run (same local machine, old JSON `/tmp/k6-smtp-*-8080.json`)

| Phase | Old err% | New err% | Delta (pp) | Old p95 (ms) | New p95 (ms) | Delta p95 (ms) |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| smoke | 3.06 | 3.00 | -0.06 | 2.86 | 5.46 | +2.60 |
| load | 19.50 | 3.45 | -16.05 | 4.48 | 6.02 | +1.54 |
| stress | 19.82 | 3.50 | -16.32 | 4.70 | 6.92 | +2.21 |
| spike | 19.73 | 3.53 | -16.20 | 176.14 | 327.37 | +151.24 |
| soak | 19.43 | 3.37 | -16.06 | 4.60 | 4.34 | -0.25 |

Interpretation:

- Error rate dropped strongly versus old SMTP-only run (about `-16pp` in load/stress/spike/soak).
- Spike latency got worse on `:8080` (higher `p95` and much higher max), while `:8081` stayed better under same profile.

### 6.2 Compare against older mixed IAM+SMTP baseline

Reference: `internal/iam/docs/loadtest/loadtest-results.md`.

- That baseline used different scenario durations (ex: load `30s`) and mixed-route matrix, so it is not apples-to-apples.
- It reported much lower error ratio (`~0.04%`) and lower p95, but under a lighter and different test profile.

## 7) Bottlenecks and risks observed

1. Stable ~`3.4%` `http_req_failed` remains across phases.
   - Main reason is expected non-2xx behavior in test mix (notably connect probes and operational transitions), which k6 still counts as failed in `http_req_failed`.
2. Spike on replica `:8080` shows latency cliff.
   - `p95` `327ms`, max `5002ms`, and `smtp_route_failure=120`.
   - Needs route-level drill-down for which write path produced unexpected statuses under spike.
3. Replica asymmetry under spike.
   - `:8081` handled the same spike profile better (`p95 151ms`, max `785ms`), indicating node-local contention/jitter rather than shared DB/Redis collapse.

## 8) Recommendation before prod

1. Keep current clean-room script and rerun with real SMTP stub alive to separate transport 5xx from true backend failures.
2. Add per-route status counters in k6 (tagged by endpoint + status code) so `http_req_failed` can be split into expected vs unexpected.
3. Reproduce spike on `:8080` with profiling enabled (CPU + heap + DB waits) to isolate the route causing `smtp_route_failure=120`.
4. Run a longer soak (`>=30m`) after spike fix to validate no latency drift and stable memory profile.
