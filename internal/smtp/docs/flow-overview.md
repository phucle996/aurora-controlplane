# SMTP Module Flow Overview

Tài liệu này mô tả các luồng xử lý chính trong module SMTP của hệ thống Antigravity Control Plane.

## 1. Resource Management Flow (CRUD & Configuration)
Luồng này quản lý các thành phần cơ bản cấu thành nên hệ thống gửi tin.

*   **Components**: Consumer, Template, Gateway, Endpoint.
*   **Flow**:
    1.  **User/Admin** tạo tài nguyên qua REST API (`transport/http/handler`).
    2.  **Service** xác thực logic nghiệp vụ và lưu trữ vào Database qua **Repository**.
    3.  Mỗi lần tài nguyên thay đổi, `runtime_version` (Sequence) sẽ được tăng lên thông qua DB Triggers.
    4.  **Data Plane** sẽ nhận thấy sự thay đổi version này trong chu kỳ Sync tiếp theo.

## 2. Control Plane <-> Data Plane Sync Flow
Đây là luồng quan trọng nhất để đảm bảo hệ thống phân tán hoạt động đồng bộ.

*   **Heartbeat**: Data Plane gửi nhịp tim định kỳ thông qua `/runtime/report` để thông báo trạng thái "sống" và công suất (capacity).
*   **Sync**:
    1.  Data Plane gọi `/runtime/sync` kèm theo `local_version` hiện tại.
    2.  Control Plane so sánh với `global_version`. Nếu có sự khác biệt, Control Plane sẽ gửi toàn bộ cấu hình mới (Consumers, Templates, Gateways, Endpoints).
*   **Report**:
    1.  Data Plane gửi báo cáo chi tiết về trạng thái các Shard, lỗi thực thi, và số lượng tin đang xử lý (`inflight_count`).
    2.  Control Plane cập nhật vào các bảng `runtime_status` để hiển thị lên Dashboard.

## 3. Sharding & Rebalance Flow (High Availability)
Đảm bảo tải được phân bổ đều và có khả năng tự phục hồi.

*   **Coordinator**: Sử dụng `gatewayCoordinator` và `consumerCoordinator` trong `RuntimeService`.
*   **Reconcile Process**:
    1.  Hệ thống kiểm tra số lượng Shard mong muốn (`desired_shard_count`).
    2.  Nếu một Data Plane bị chết (quá hạn heartbeat), Coordinator sẽ thực hiện **Rebalance**.
    3.  Các Shard sẽ được chuyển trạng thái từ `active` -> `revoking` -> `pending` -> `active` trên các node mới.
    4.  Sử dụng cơ chế `lease` để đảm bảo không có 2 node cùng xử lý 1 shard tại một thời điểm.

## 4. Message Delivery Flow (Conceptual)
Mô tả cách một email được xử lý (Logic phối hợp).

1.  **Source** (API/Queue) đẩy tin vào **Consumer**.
2.  Consumer sử dụng **Template** để render nội dung (Subject/Body).
3.  Hệ thống định tuyến tin qua **Gateway** dựa trên `TrafficClass` (Transactional/Marketing).
4.  Gateway chọn **Endpoint** (SMTP/SES/Mailgun) tối ưu nhất (dựa trên Priority/Weight) để gửi đi.
5.  Kết quả (Success/Fail) được ghi vào **Delivery Attempts**.

## 5. Observability & Aggregation Flow
Luồng dữ liệu phục vụ giám sát và Dashboard.

*   **Aggregation**:
    1.  `AggregationRepository` sử dụng các câu truy vấn tối ưu (CTE, FILTER) để quét bảng `delivery_attempts` và `activity_logs`.
    2.  Dữ liệu được lọc theo `workspace_id` để đảm bảo tính đa người dùng (Multi-tenancy).
*   **Timeline**:
    1.  Mọi hành động quan trọng được ghi lại vào `activity_logs`.
    2.  Dashboard hiển thị 10 hành động gần nhất của Workspace hiện tại.
*   **Metrics**: Hiển thị throughput (7 ngày), tỷ lệ lỗi, và trạng thái sức khỏe của các Gateway theo thời gian thực.

---
*Cập nhật lần cuối: 22/04/2026*
