# SMTP Prod Readiness Review

Pham vi danh gia:
- `controlplane/internal/smtp`
- `dataplane/internal/smtp`

Ngay danh gia:
- `2026-04-22`

## Tong ket nhanh

- Diem san sang cho prod: `70/100`
- Control plane: `74/100`
- Data plane: `66/100`

Nhan xet nhanh:
- Kien truc tach lop kha tot, cache/adapter design kha ro rang, co mTLS cho giao tiep runtime, co migration + index + trigger.
- Nhung module dang co 4 vung rui ro lon: secret exposure, cache stale khi config doi, metric aggregation sai/ton DB, va adapter Kafka khong xu ly het batch record.

## Diem manh

- Control plane tuan thu kha sat layer `handler -> service -> repository`.
- SQL duoc dat trong repository, co mapping entity/model ro rang.
- Co bo index kha day du cho gateway/template/endpoint/delivery/runtime tables.
- Runtime gRPC co xac thuc peer bang chung chi va kiem tra dataplane identity.
- Secret duoc tach bang rieng (`consumer_secrets`, `endpoint_secrets`), co y tuong provider/ref/version.
- Co background reconcile, co heartbeat/report/sync, co observability dashboard.

## Finding uu tien cao

### 1. Secret dang bi luu plaintext va co the bi tra ve qua API

**Muc do:** High

**Anh huong:**
- `smtp.endpoint_secrets` luu `password`, `client_key_pem`, `client_cert_pem`, `ca_cert_pem` duoi dang text thuan.
- API `GET/POST/PUT endpoint` dang tra ve cac truong private key/cert cho client.
- Neu role read bi lo, user co the lay duoc secret material thay vi chi metadata.

**Bang chung:**
- `controlplane/internal/smtp/migrations/000002_create_tables.up.sql`
- `controlplane/internal/smtp/transport/http/handler/endpoint_handler.go`
- `controlplane/internal/smtp/repository/endpoint_repo.go`

**Vi du doan nguy hiem:**
- `controlplane/internal/smtp/transport/http/handler/endpoint_handler.go:126-146`
- `controlplane/internal/smtp/transport/http/handler/endpoint_handler.go:297-317`
- `controlplane/internal/smtp/transport/http/handler/endpoint_handler.go:400-420`

**De xuat:**
- Khong tra secret material trong response thong thuong.
- Tach flow `metadata view` va `secret edit/fetch` rieng.
- Chuyen sang secret backend/envelope encryption, hoac toi thieu encrypt at rest cho private key/password.

### 2. Dataplane cache adapter/pool theo ID nen config update khong co tac dung ngay

**Muc do:** High

**Anh huong:**
- `ExecutionService.ApplyBundle()` giu lai `runtime.adapter` neu adapter da ton tai.
- `RawSMTPSender.pool()` cache theo `endpoint.ID` va khong invalidate khi `RuntimeVersion` / `SecretVersion` doi.
- Ket qua: doi credential, host, TLS mode, broker config nhung node van dung config cu cho den khi restart hoac shard remap.

**Bang chung:**
- `dataplane/internal/smtp/service/execution_service.go:155-177`
- `dataplane/internal/smtp/adapter/smtp_sender.go:150-186`

**De xuat:**
- Key cache theo `ID + RuntimeVersion` hoac hash cua config/secret version.
- Khi bundle moi co config moi, clear pool cu va recreate adapter.
- Them test cho truong hop rotate secret / doi host / doi broker.

### 3. Query aggregation bi multiply row, lam sai metric va ton DB

**Muc do:** High

**Anh huong:**
- `FULL OUTER JOIN` giua gateways va templates, sau do `LEFT JOIN` consumers/statuses, co the nhan nhieu row trung lap.
- `SUM(crs.broker_lag)` co the bi phong dai, trong khi query chay tren dashboard hot path.
- Cung 1 query dang gop metric, throughput, health, queue mix, nen chi phi tang theo workspace size.

**Bang chung:**
- `controlplane/internal/smtp/repository/aggregation_repo.go:24-45`
- `controlplane/internal/smtp/repository/aggregation_repo.go:48-192`

**De xuat:**
- Tach metrics thanh cac subquery/CTE doc lap.
- Khong join chéo gateways/templates/consumers khi muon tong hop queue.
- Can nhac materialized view hoac pre-aggregate bang job neu dashboard load lon.

### 4. Runtime sync/report dang qua dat cho hot path

**Muc do:** High

**Anh huong:**
- Moi `Report()` co the keo theo `Reconcile()` va `buildSyncResponse()`.
- `buildSyncResponse()` doc assignments roi bat dau N+1 query cho consumers/gateways/endpoints/templates.
- `ReplaceGatewayStatuses()` va `ReplaceConsumerStatuses()` dung `DELETE + INSERT` moi lan report, tao write amplification.

**Bang chung:**
- `controlplane/internal/smtp/service/runtime_service.go:79-145`
- `controlplane/internal/smtp/service/runtime_service.go:169-278`
- `controlplane/internal/smtp/repository/runtime_repo.go:184-247`

**De xuat:**
- Chi hydrate full config khi version that su doi.
- Batch load theo danh sach ID thay vi query tung entity.
- Giam write amplification cho status tables, uu tien batch upsert hoac partial update.
- Tach reconcile cadence khoi heartbeat/report cadence neu can.

### 5. Kafka adapter chi lay record dau tien trong moi fetch batch

**Muc do:** Medium

**Anh huong:**
- `Consume()` goi `PollFetches()` nhung chi giu record dau tien va bo qua cac record con lai trong batch.
- Dieu nay lam throughput Kafka thap hon can thiet va co nguy co bo sot record trong app-level processing.

**Bang chung:**
- `dataplane/internal/smtp/adapter/kafka.go:73-101`

**De xuat:**
- Buffer toan bo record trong batch, hoac chi poll 1 record mot lan neu muon semantics don gian.
- Bao ve timing/offset handling bang regression test.

### 6. Owner cua gateway dang do client cung cap

**Muc do:** Medium

**Anh huong:**
- `owner_user_id` cua gateway dang lay truc tiep tu request body, khac voi consumer/template/endpoint la lay tu auth context.
- Co the lam sai audit trail hoac spoof ownership neu client gui gia tri khac.

**Bang chung:**
- `controlplane/internal/smtp/transport/http/handler/gateway_handler.go:460-473`
- `controlplane/internal/smtp/transport/http/handler/gateway_handler.go:585-599`

**De xuat:**
- Lay owner tu `middleware.GetUserID(c)` hoac server-side identity.
- Neu muon support assign owner tu client, can validate role va audit them.

## Diem so theo khia canh

- Architecture va layering: `86/100`
- Security: `64/100`
- Performance va scale: `62/100`
- Flexibility va extensibility: `68/100`
- Operability va reliability: `70/100`

## De xuat update uu tien

### P0

1. Loai bo secret material khoi HTTP response va chuyen sang secret management ro rang.
2. Invalidate cache adapter/pool theo `RuntimeVersion` / `SecretVersion`.
3. Sua aggregation query thanh cac subquery doc lap de tranh overcount.

### P1

1. Batch hoa runtime sync/report, tranh N+1 va full workspace scan tren moi heartbeat.
2. Sua Kafka adapter de xu ly het record trong fetch batch.
3. Chuan hoa owner attribution cua gateway ve server-side identity.

### P2

1. Can nhac pagination cho `activity_logs` va `delivery_attempts` khi dashboard tang lon.
2. Them rate limit / egress control cho cac `try-connect` endpoint.
3. Neu muon scale manh, can nhac materialized view cho dashboard metrics.

## Verification

Da chay:
- `go test ./internal/smtp/...` trong `dataplane` -> pass
- `go test ./internal/smtp/...` trong `controlplane` -> pass tren workspace hien tai

Ghi chu:
- `security.Claims.TenantID` la field exported trong `controlplane/internal/security/jwt.go`, nen `claims.TenantID` o `access.go` khong bi anh huong boi `omitempty`.
- Loi `claims.TenantID undefined` neu xuat hien o snapshot cu thi co the la stale build hoac khac revision; khong reproduce tren code hien tai.

## Ket luan

Module SMTP co nen tang architecture tot va co kha nang scale o muc trung binh-kha, nhung chua dat muc "production hardening" cao do 4 van de lon:
- secrets exposure,
- cache stale khi config doi,
- metric query sai/ton,
- va hot path sync/report qua nang.

Neu fix 4 diem nay truoc, diem prod co the nang len khoang `80+/100`.
