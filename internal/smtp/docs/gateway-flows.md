# Gateway Flows Documentation

Nhóm này mô tả các luồng xử lý liên quan đến Gateway - thành phần điều phối và định tuyến tin nhắn chính trong hệ thống SMTP.

---

## Flow 1: Gateway Provisioning (Create/Update)
**Mô tả**: Người dùng tạo mới hoặc cập nhật cấu hình Gateway (Traffic Class, Priority, Shard Count).

### Use Case
Admin cấu hình một Gateway mới để phục vụ luồng gửi tin "Transactional" với mức ưu tiên cao nhất.

### Sequence Diagram
```mermaid
sequenceDiagram
    participant User as Admin
    participant H as Gateway Handler
    participant S as Gateway Service
    participant R as Gateway Repository
    participant SR as Shard Logic (syncGatewayShards)
    participant DB as PostgreSQL

    User->>H: POST /api/v1/smtp/gateways
    H->>S: CreateGateway(ctx, gateway)
    S->>R: SaveGateway(ctx, gateway)
    R->>DB: BEGIN TX
    R->>DB: INSERT INTO smtp.gateways
    R->>SR: syncGatewayShards(gateway_id, shard_count)
    SR->>DB: DELETE extra shards / INSERT missing shards
    R->>DB: COMMIT TX
    S-->>H: Success
    H-->>User: 201 Created
```

### Tech Lead Spec
*   **Shard Pre-allocation**: Khác với các tài nguyên khác, khi tạo Gateway, hệ thống phải khởi tạo ngay các slot trong bảng `smtp.gateway_shards` để bộ điều phối (Coordinator) có thể bắt đầu gán node.
*   **Runtime Version**: Cập nhật Gateway sẽ làm tăng version toàn cục, buộc các Data Plane phải load lại routing table.

---

## Flow 2: Gateway Endpoint Mapping & Priority
**Mô tả**: Gán các Endpoint (hạ tầng gửi tin) vào Gateway và thiết lập mức độ ưu tiên.

### Use Case
Người dùng gán 2 Endpoint SMTP vào 1 Gateway: 1 cái là Main (Priority 1), 1 cái là Backup (Priority 2).

### Sequence Diagram
```mermaid
sequenceDiagram
    participant User as Admin
    participant H as Gateway Handler
    participant R as Gateway Repository
    participant DB as PostgreSQL

    User->>H: PUT /api/v1/smtp/gateways/{id}/endpoints
    H->>R: UpdateGatewayEndpoints(gateway_id, endpoint_ids, weights)
    R->>DB: DELETE FROM smtp.gateway_endpoints WHERE gateway_id = ?
    R->>DB: INSERT INTO smtp.gateway_endpoints (multi-rows)
    DB-->>R: Success
    H-->>User: 200 OK
```

### Tech Lead Spec
*   **Routing Metadata**: Dữ liệu này được lưu trong bảng trung gian `smtp.gateway_endpoints`.
*   **Cache Invalidation**: Việc thay đổi mapping này không làm thay đổi `runtime_version` của bản thân Gateway (trừ khi có trigger hỗ trợ), nên cần lưu ý cơ chế cập nhật tại node thực thi.

---

## Flow 3: Gateway Fallback Logic
**Mô tả**: Cách hệ thống tự động chuyển vùng khi Gateway chính gặp sự cố.

### Use Case
Gateway "High-Speed" bị lỗi hoặc quá tải, hệ thống tự động định tuyến tin nhắn sang Gateway "Slow-Backup".

### Sequence Diagram
```mermaid
sequenceDiagram
    participant DP as Data Plane Engine
    participant G1 as Primary Gateway
    participant G2 as Fallback Gateway
    participant E as Endpoint

    DP->>G1: Attempt Delivery
    G1->>E: Send Request
    E-->>G1: 5xx Error / Connection Timeout
    G1->>G1: Check Fallback Gateway ID
    G1->>G2: Route Message to Fallback
    G2->>E: Send Request (via Backup Endpoint)
    E-->>G2: 250 OK
```

### Tech Lead Spec
*   **Recursive Check**: Logic fallback có thể hỗ trợ nhiều cấp (G1 -> G2 -> G3).
*   **Metric Impact**: Khi xảy ra fallback, `activity_logs` cần ghi nhận để Admin có thể phát hiện hạ tầng chính đang gặp vấn đề.
