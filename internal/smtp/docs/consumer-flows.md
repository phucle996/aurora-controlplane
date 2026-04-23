# Consumer Flows Documentation

Nhóm này mô tả cách hệ thống cấu hình và quản lý các Consumer - các "đầu nạp" dữ liệu từ Kafka, RabbitMQ hoặc các hàng chờ khác.

---

## Flow 1: Consumer Configuration & Secret Management
**Mô tả**: Thiết lập thông tin kết nối và lưu trữ thông tin nhạy cảm (Password/Key) an toàn.

### Use Case
Admin cấu hình một Kafka Consumer để đọc tin nhắn từ topic `email_notifications`.

### Sequence Diagram
```mermaid
sequenceDiagram
    participant User as Admin
    participant H as Consumer Handler
    participant S as Consumer Service
    participant R as Consumer Repository
    participant DB as PostgreSQL

    User->>H: POST /api/v1/smtp/consumers
    H->>S: CreateConsumer(ctx, req)
    S->>R: SaveWithSecrets(ctx, consumer, secret_data)
    R->>DB: BEGIN TX
    R->>DB: INSERT INTO smtp.consumers
    R->>DB: INSERT INTO smtp.consumer_secrets (Encrypt config)
    R->>DB: COMMIT TX
    S-->>H: Success
    H-->>User: 201 Created
```

### Tech Lead Spec
*   **Secret Separation**: Thông tin nhạy cảm (`consumer_secrets`) được tách riêng bảng với cấu hình chung để áp dụng các lớp bảo mật và mã hóa khác nhau.
*   **Transport Types**: Hỗ trợ đa dạng giao thức (`kafka`, `pubsub`, `webhook`) định nghĩa qua Enum `smtp.consumer_transport_type`.

---

## Flow 2: Consumer Runtime Scaling (Sharding)
**Mô tả**: Điều chỉnh số lượng worker thực tế xử lý hàng chờ của Consumer.

### Use Case
Hàng chờ email bị nghẽn, Admin tăng `desired_shard_count` từ 2 lên 5 để tăng tốc độ xử lý.

### Sequence Diagram
```mermaid
sequenceDiagram
    participant User as Admin
    participant H as Consumer Handler
    participant S as Consumer Service
    participant R as Consumer Repository
    participant Coord as Rebalance Coordinator
    participant DB as PostgreSQL

    User->>H: PATCH /api/v1/smtp/consumers/{id} (shard_count: 5)
    H->>S: UpdateShardCount(id, 5)
    S->>R: UpdateDesiredShards(id, 5)
    R->>DB: UPDATE smtp.consumers SET desired_shard_count = 5
    Note over S, Coord: Background Reconcile Loop
    Coord->>DB: Check actual vs desired
    Coord->>DB: INSERT 3 new shard slots
    S-->>H: Success
    H-->>User: 200 OK
```

### Tech Lead Spec
*   **Horizontal Scaling**: Mỗi Shard tương ứng với một đơn vị xử lý song song. Việc tăng shard count sẽ kích hoạt Coordinator phân bổ thêm node Data Plane vào luồng xử lý này.
*   **Lag Monitoring**: Số lượng shard nên được điều chỉnh dựa trên chỉ số `lag` được báo cáo từ Data Plane.
