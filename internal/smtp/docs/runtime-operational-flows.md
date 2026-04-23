# Runtime & Operational Flows Documentation

Nhóm này mô tả các luồng vận hành hệ thống, bao gồm đồng bộ trạng thái node và tổng hợp dữ liệu Dashboard.

---

## Flow 1: Node Synchronization (Config Convergence)
**Mô tả**: Đảm bảo tất cả các node Data Plane đều chạy với cấu hình mới nhất từ Control Plane.

### Use Case
Admin vừa cập nhật Template, Data Plane cần nhận diện thay đổi này để render email chính xác.

### Sequence Diagram
```mermaid
sequenceDiagram
    participant DB as PostgreSQL
    participant CP as Control Plane
    participant DP as Data Plane Node

    Note over DB: User updates Template (Version 100 -> 101)
    loop Mỗi 15s (Sync Cycle)
        DP->>CP: GET /runtime/sync (current_version: 100)
        CP->>DB: SELECT max(runtime_version)
        DB-->>CP: 101
        CP->>DB: SELECT full_config WHERE version > 100
        DB-->>CP: Updated Template Data
        CP-->>DP: 200 OK (Payload: Template 101)
        DP->>DP: Reload Local Cache
    end
```

### Tech Lead Spec
*   **Eventual Consistency**: Hệ thống đạt trạng thái đồng nhất sau tối đa một chu kỳ Sync.
*   **Payload Optimization**: Sử dụng Gzip compression cho Sync Payload khi số lượng tài nguyên lớn.

---

## Flow 2: Dashboard Data Aggregation
**Mô tả**: Tổng hợp log gửi tin để hiển thị biểu đồ và chỉ số hiệu năng.

### Use Case
Người dùng mở Dashboard xem tỷ lệ gửi thành công trong 24h qua.

### Sequence Diagram
```mermaid
sequenceDiagram
    participant User as Browser
    participant H as Aggregation Handler
    participant R as Aggregation Repository
    participant DB as PostgreSQL

    User->>H: GET /api/v1/smtp/aggregation
    H->>R: GetWorkspaceAggregation(workspace_id)
    R->>DB: SELECT count(1) FILTER(...) FROM delivery_attempts
    Note right of DB: Optimized Composite Index Scan
    DB-->>R: Aggregated Rows (Today, Throughput, Health)
    R-->>H: entity.SMTPOverview
    H-->>User: 200 OK (JSON Data)
```

### Tech Lead Spec
*   **Query Performance**: Truy vấn sử dụng các hàm `FILTER` và `date_trunc` để tránh phải tính toán ở tầng Application.
*   **Index Usage**: Bắt buộc sử dụng Index `(workspace_id, created_at DESC)` để duy trì tốc độ < 100ms.
