# Backlog 3 — Client xem và lọc danh sách cảnh báo

Backlog 3 cho phép client truy vấn lịch sử alert đã được lưu (do device gửi qua Backlog 2), với các bộ lọc theo thiết bị, mức độ nghiêm trọng và khoảng thời gian. Backlog 5 mở rộng cùng endpoint này bằng `search` nhưng giữ nguyên response shape và pagination.

> Yêu cầu: Client xem/lọc danh sách cảnh báo theo thiết bị, mức độ nghiêm trọng và thời gian.

---

## Mục tiêu đã triển khai

| Hạng mục | Trạng thái |
| --- | --- |
| Client xem được danh sách alert thuộc về mình | Hoàn thành |
| Mọi truy vấn scope theo `client_id` từ JWT | Hoàn thành |
| Filter theo `device_id` | Hoàn thành |
| Filter theo `severity` đơn và đa giá trị | Hoàn thành |
| Filter theo `from`/`to` (RFC3339) trên `occurred_at` | Hoàn thành |
| Pagination `page`/`page_size` (default 20, max 100) | Hoàn thành |
| Search theo message/type/tên device/exact device UUID qua Backlog 5 | Hoàn thành |
| Order timeline mới nhất trước (`occurred_at DESC, id DESC`) | Hoàn thành |
| Validation rõ ràng cho mọi tham số | Hoàn thành |
| Không leak alert sang client khác | Hoàn thành |
| Swagger có request/response để reviewer test | Hoàn thành |

---

## Acceptance criteria

| Tiêu chí | Cách kiểm tra |
| --- | --- |
| API `/alerts` yêu cầu client JWT | Gọi không có Authorization → `401 Unauthorized` |
| Client lấy được lịch sử alert của mình | `GET /alerts` mặc định trả alert + pagination meta |
| Cross-client isolation | Client A không thấy alert Client B kể cả khi truyền `device_id` của B |
| Filter device đúng | `?device_id=...` chỉ trả alert của device đó |
| Filter multi-severity đúng | `?severity=warning&severity=critical` chỉ trả 2 mức |
| Filter time range đúng | `?from=...&to=...` chỉ trả alert trong khoảng |
| Search compose đúng | `?search=smoke&severity=critical&device_id=...` chỉ trả alert match tất cả điều kiện |
| Validation severity | severity ngoài enum → `400 INVALID_SEVERITY` |
| Validation time | format sai → `400 INVALID_TIME_FORMAT`; from > to → `400 INVALID_TIME_RANGE` |
| Validation pagination | `page < 1` hoặc `page_size > 100` → `400 INVALID_PAGINATION` |
| Validation device_id | UUID sai → `400 INVALID_DEVICE_ID` |
| Order ổn định | Alert mới nhất xuất hiện trước; tie-break theo `id DESC` |
| Response không lộ `client_id` | JSON item không chứa `client_id` |

---

## Tổng quan luồng xử lý

```text
Client (JWT)
  │  GET /api/v1/alerts?device_id=...&severity=warning
  │       &from=...&to=...&search=smoke&page=1&page_size=20
  │  Authorization: Bearer <access_token>
  ▼
Auth middleware
  │  set client_id vào context từ JWT
  ▼
QueryHandler.List
  │  parse query → ListAlertsInput
  ▼
QueryService.ListAlerts
  │  validate severity, time range, pagination, device_id, search
  │  build ListFilter
  ▼
AlertRepository.List
  │  SELECT ... FROM alerts
  │  WHERE client_id = $1 AND optional filters/search
  │  ORDER BY occurred_at DESC, id DESC
  │  LIMIT/OFFSET; + COUNT(*) cho total
  ▼
Response
  data[]: AlertResponse{id, device_id, type, severity,
                        message, payload, occurred_at, received_at}
  pagination: {page, page_size, total, total_pages,
               has_next, has_previous}
```

---

## Authentication cho Backlog 3

API `/alerts` yêu cầu client JWT access token:

```http
Authorization: Bearer <access_token>
```

Token lấy từ:

```http
POST /api/v1/auth/login
```

Demo client ở môi trường development:

```text
Email: client@example.com
Password: password123
```

Backlog 3 chỉ dùng client JWT. Device API key không dùng cho endpoint này.

---

## API đã triển khai

Base URL:

```text
http://localhost:8080/api/v1
```

| Method | Endpoint | Mục đích | Auth |
| --- | --- | --- | --- |
| GET | `/alerts` | Lấy danh sách alert của client với filter và pagination | Client JWT |

---

## Query parameters

Tất cả tham số đều optional. Mặc định trả mọi alert thuộc client hiện tại theo trang đầu tiên.

| Tham số | Kiểu | Mô tả |
| --- | --- | --- |
| `device_id` | UUID | Lọc theo 1 thiết bị cụ thể của client |
| `severity` | enum, lặp được | `info`, `warning`, hoặc `critical`. Lặp để chọn nhiều mức: `?severity=warning&severity=critical` |
| `from` | RFC3339 | Lower bound trên `occurred_at` (>=) |
| `to` | RFC3339 | Upper bound trên `occurred_at` (<=) |
| `search` | string 2..100 | Tìm case-insensitive trong message, type, tên device hoặc exact device UUID |
| `page` | int >= 1 | Trang hiện tại, mặc định 1 |
| `page_size` | int 1..100 | Kích thước trang, mặc định 20 |

`from`/`to` dùng `occurred_at` (thời điểm event xảy ra ở device), không dùng `received_at`. `search` là phần mở rộng của Backlog 5: blank sau khi trim sẽ bị bỏ qua, còn search 1 ký tự hoặc trên 100 ký tự trả `400 INVALID_SEARCH`.

---

## Schema response

### Alert item

| Field | Ý nghĩa |
| --- | --- |
| `id` | UUID của alert |
| `device_id` | UUID của device đã gửi alert |
| `type` | Loại cảnh báo, do device đặt |
| `severity` | `info`, `warning`, hoặc `critical` |
| `message` | Nội dung cảnh báo |
| `payload` | Metadata JSON tự do từ device |
| `occurred_at` | Thời điểm event xảy ra ở device |
| `received_at` | Thời điểm server nhận và lưu event |

Response không bao giờ chứa `client_id` hoặc raw device API key.

### Pagination meta

| Field | Ý nghĩa |
| --- | --- |
| `page` | Trang hiện tại |
| `page_size` | Kích thước trang |
| `total` | Tổng số alert match filter |
| `total_pages` | Tổng số trang |
| `has_next` | Có trang tiếp theo hay không |
| `has_previous` | Có trang trước hay không |

---

## Luồng test chính cho reviewer

### 1. Login lấy access token

```http
POST /api/v1/auth/login
Content-Type: application/json
```

Request:

```json
{
  "email": "client@example.com",
  "password": "password123"
}
```

Lưu `data.access_token` và Authorize Swagger với `Bearer <access_token>`.

---

### 2. List mặc định

```http
GET /api/v1/alerts
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

```json
{
  "status": true,
  "message": "Alerts retrieved successfully",
  "data": [
    {
      "id": "9f3d2e1a-...",
      "device_id": "4d285f4b-...",
      "type": "high_temperature",
      "severity": "critical",
      "message": "...",
      "payload": { "...": "..." },
      "occurred_at": "2026-05-04T12:00:00Z",
      "received_at": "2026-05-04T12:00:00.123Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 111,
    "total_pages": 6,
    "has_next": true,
    "has_previous": false
  }
}
```

---

### 3. Filter theo device

```http
GET /api/v1/alerts?device_id={device_id}
Authorization: Bearer <access_token>
```

Response: chỉ alert của device đó.

---

### 4. Filter theo severity (đơn và đa giá trị)

```http
GET /api/v1/alerts?severity=warning
GET /api/v1/alerts?severity=warning&severity=critical
Authorization: Bearer <access_token>
```

Response: chỉ alert có severity match.

---

### 5. Filter theo time range

```http
GET /api/v1/alerts?from=2026-05-01T00:00:00Z&to=2026-05-04T23:59:59Z
Authorization: Bearer <access_token>
```

Response: chỉ alert có `occurred_at` nằm trong khoảng.

---

### 6. Pagination

```http
GET /api/v1/alerts?page=2&page_size=5
Authorization: Bearer <access_token>
```

Response: trang 2 với tối đa 5 item, `has_next` và `has_previous` phản ánh đúng vị trí.

---

### 7. Kết hợp nhiều filter

```http
GET /api/v1/alerts?device_id={id}&severity=critical&from=2026-05-04T00:00:00Z&page_size=10
Authorization: Bearer <access_token>
```

Response: alert match đủ tất cả điều kiện.

---

## Negative cases nên kiểm tra

| Case | Kết quả mong đợi |
| --- | --- |
| Không có access token | `401 Unauthorized` |
| Access token sai/expired | `401 Unauthorized` |
| `device_id` không phải UUID | `400 INVALID_DEVICE_ID` |
| `severity` ngoài enum (`urgent`, ...) | `400 INVALID_SEVERITY` |
| `from`/`to` sai format | `400 INVALID_TIME_FORMAT` |
| `from > to` | `400 INVALID_TIME_RANGE` |
| `page < 1` | `400 INVALID_PAGINATION` |
| `page_size > 100` hoặc `< 1` | `400 INVALID_PAGINATION` |
| `device_id` của client khác | `200` với `data: []` và `total: 0` (không leak) |

---

## Business rules

- Mọi truy vấn scope theo `client_id` từ JWT — không bao giờ trả alert của client khác.
- Filter `device_id` chỉ áp dụng nếu device thuộc client hiện tại; ngược lại trả empty thay vì 404 để không tiết lộ existence của device thuộc client khác.
- `severity` là enum, lặp để chọn nhiều mức; giá trị duplicate được dedup ở service.
- `from`/`to` filter trên `occurred_at`, không phải `received_at`.
- Order cố định `occurred_at DESC, id DESC` để timeline ổn định và mới nhất hiển thị trước.
- Pagination hard cap 100 cho `page_size`.
- Response item không chứa `client_id`; cũng không chứa raw device API key (bảng `alerts` không lưu key).
- Alert là append-only: API hiện tại không có endpoint update/delete alert.
- Backlog 4 có thể append alert `type="auto_escalated"`, `severity="critical"`; client có thể thấy alert này trong `/alerts`, payload có `source_alert_ids` chứa đầy đủ alert IDs nguồn trong rolling window, và client có thể tự filter theo `type` nếu cần.

---

## Checklist reviewer cho Backlog 3

- [ ] Chạy `make dev-up` thành công.
- [ ] Login `client@example.com / password123` để lấy access token.
- [ ] Tạo device và gửi vài alert (qua Backlog 2) để có dữ liệu test.
- [ ] `GET /api/v1/alerts` mặc định trả alert có pagination meta đầy đủ.
- [ ] `GET /api/v1/alerts?device_id=...` chỉ trả alert của device đó.
- [ ] `GET /api/v1/alerts?severity=warning` chỉ trả alert warning.
- [ ] `GET /api/v1/alerts?severity=warning&severity=critical` trả 2 mức.
- [ ] `GET /api/v1/alerts?from=...&to=...` lọc đúng theo time range.
- [ ] `GET /api/v1/alerts?page=2&page_size=5` trả đúng trang.
- [ ] Gọi không có access token → `401 Unauthorized`.
- [ ] Gọi với severity sai → `400 INVALID_SEVERITY`.
- [ ] Gọi với time format sai → `400 INVALID_TIME_FORMAT`.
- [ ] Gọi với from > to → `400 INVALID_TIME_RANGE`.
- [ ] Gọi với page_size > 100 → `400 INVALID_PAGINATION`.
- [ ] Gọi với device_id sai UUID → `400 INVALID_DEVICE_ID`.
- [ ] Login client khác → không thấy alert của client cũ.

---

## Mapping yêu cầu đề bài sang implementation

| Yêu cầu đề bài | Implementation |
| --- | --- |
| Client xem danh sách cảnh báo | `GET /api/v1/alerts` |
| Lọc theo thiết bị | Query `device_id` |
| Lọc theo mức độ nghiêm trọng | Query `severity` (lặp được, multi-value) |
| Lọc theo thời gian | Query `from`/`to` trên `occurred_at` |
| Bảo vệ dữ liệu giữa các client | Repository scope `WHERE client_id = $1` |
| Pagination cho dữ liệu lớn | Query `page`/`page_size`, hard cap 100 |
| Order timeline | `ORDER BY occurred_at DESC, id DESC` |

---

## Cách chạy local để test Backlog 3

Khởi động môi trường:

```bash
make dev-up
```

Mở Swagger:

```text
http://localhost:8080/swagger/index.html
```

Test bằng curl:

```bash
ACCESS=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"client@example.com","password":"password123"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")

curl -s "http://localhost:8080/api/v1/alerts?severity=warning&severity=critical&page_size=5" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Chạy kiểm tra code:

```bash
go test ./...
go build ./...
go vet ./...
```
