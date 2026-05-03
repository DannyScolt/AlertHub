# Backlog 2 — Device gửi realtime alert event

Backlog 2 tập trung vào việc device gửi event cảnh báo realtime lên hệ thống, API lưu alert vào PostgreSQL và fan-out realtime cho client qua SSE stream.

> Yêu cầu: Thiết bị gửi event cảnh báo realtime lên hệ thống.

---

## Mục tiêu đã triển khai

| Hạng mục | Trạng thái |
| --- | --- |
| Device gửi một alert event bằng device API key | Hoàn thành |
| Device gửi batch alert events, tối đa 100 events/request | Hoàn thành |
| API validate type, severity, message và payload | Hoàn thành |
| Alert được lưu append-only vào PostgreSQL | Hoàn thành |
| API phát PostgreSQL NOTIFY sau khi lưu alert | Hoàn thành |
| Alert listener nhận NOTIFY và fan-out tới SSE subscribers | Hoàn thành |
| Client mở SSE stream bằng JWT access token | Hoàn thành |
| SSE stream chỉ trả alert thuộc client hiện tại | Hoàn thành |
| SSE stream có thể lọc theo device_id | Hoàn thành |
| SSE stream gửi heartbeat định kỳ | Hoàn thành |
| Soft-deleted device không gửi event được | Hoàn thành |
| Swagger mô tả rõ client JWT và device API key auth | Hoàn thành |

---

## Tổng quan luồng xử lý

Backlog 2 dùng hai loại authentication khác nhau:

```text
Client JWT access token
└── Dùng để client mở realtime stream: GET /alerts/stream

Device API key
└── Dùng để device gửi event: POST /events hoặc POST /events/batch
```

Luồng realtime chính:

```text
Device
  │
  │ POST /api/v1/events
  │ Authorization: Bearer ah_dev_xxx
  ▼
Event Ingest API
  │
  │ validate + resolve device từ API key
  ▼
Alert Service
  │
  │ insert alert
  ▼
PostgreSQL alerts
  │
  │ pg_notify
  ▼
Alert Listener
  │
  │ fan-out theo client_id / device_id
  ▼
SSE Stream
  │
  │ event: alert
  ▼
Client Dashboard
```

---

## Thành phần liên quan

```text
devices
└── Lưu device API key hash và trạng thái device. Device đã soft-delete không được gửi event.

alerts
└── Lưu alert event append-only, gồm device_id, client_id, type, severity, message, payload, occurred_at và received_at.

PostgreSQL LISTEN/NOTIFY
└── Dùng để phát tín hiệu realtime sau khi alert được lưu.

SSE Stream
└── Giữ kết nối HTTP mở để client nhận alert realtime.
```

Alert là append-only event: API hiện tại không update/delete alert.

---

## Acceptance criteria

Reviewer có thể xem Backlog 2 là hoàn thành khi các điều kiện bên dưới đều đúng:

| Tiêu chí | Cách kiểm tra |
| --- | --- |
| Device gửi một alert hợp lệ thành công | `POST /events` trả `202 Accepted` và `alert_id` |
| Device gửi batch events thành công | `POST /events/batch` trả số lượng `accepted/rejected` đúng |
| Batch hỗ trợ partial success | Event hợp lệ được lưu, event lỗi nằm trong `errors` theo `index` |
| Batch giới hạn tối đa 100 events | Batch rỗng hoặc quá 100 events bị reject |
| Device auth tách biệt client auth | `/events` dùng device API key, không dùng client JWT |
| SSE stream dùng client JWT | `/alerts/stream` không chấp nhận device API key |
| Client nhận alert realtime qua SSE | Mở stream rồi gửi event sẽ thấy `event: alert` |
| Stream gửi connected event ban đầu | Kết nối SSE trả `event: connected` |
| Stream gửi heartbeat định kỳ | Sau khoảng 30 giây có `event: heartbeat` |
| Stream filter theo device_id đúng | Query `device_id` chỉ nhận alert của device đó |
| Không lộ dữ liệu cross-client | Client A không nhận alert của Client B |
| Device đã soft-delete không gửi event được | Event request trả `401 Unauthorized` |
| Alert được lưu append-only | Alert insert vào bảng `alerts`, không update/delete |
| `last_seen_at` của device được tính từ alert | Sau khi gửi alert, device detail/list có thời điểm seen mới |
| Swagger đủ để reviewer tự test | Swagger có `DeviceAPIKey` và `BearerAuth` rõ ràng |

---

## Authentication cho Backlog 2

### Device gửi event

Device dùng raw device API key:

```http
Authorization: Bearer ah_dev_xxx
```

Device API key được trả về một lần khi:

- Tạo device bằng `POST /api/v1/devices`.
- Rotate key bằng `POST /api/v1/devices/{id}/rotate-key`.

API chỉ lưu hash của device API key, không lưu raw key.

### Client nhận realtime alert

Client dùng JWT access token:

```http
Authorization: Bearer <access_token>
```

Token lấy từ:

```http
POST /api/v1/auth/login
```

---

## Alert Severity

Các severity hợp lệ:

```text
info
warning
critical
```

Nếu gửi severity khác, API trả lỗi validation/business error.

---

## Dữ liệu request/response quan trọng

### Single event request

| Field | Kiểu | Bắt buộc | Ghi chú |
| --- | --- | --- | --- |
| `type` | string | Có | Loại cảnh báo, free string, không rỗng, tối đa 100 ký tự |
| `severity` | string enum | Có | Một trong `info`, `warning`, `critical` |
| `message` | string | Có | Nội dung cảnh báo, không rỗng |
| `payload` | object | Không | Metadata JSON tự do từ device |
| `occurred_at` | datetime | Không | Thời điểm event xảy ra ở device; nếu không gửi thì server dùng thời gian hiện tại |

### Single event response

| Field | Ý nghĩa |
| --- | --- |
| `alert_id` | UUID của alert vừa được lưu |
| `received_at` | Thời điểm server nhận và lưu event |

### Batch request

| Field | Kiểu | Bắt buộc | Ghi chú |
| --- | --- | --- | --- |
| `events` | array | Có | Danh sách event, tối đa 100 item |

Mỗi item trong `events` dùng cùng schema với single event request.

### Batch response

| Field | Ý nghĩa |
| --- | --- |
| `accepted` | Số event hợp lệ đã được lưu |
| `rejected` | Số event bị reject |
| `alerts` | Danh sách item thành công, gồm `index` và `alert_id` |
| `errors` | Danh sách item lỗi, gồm `index`, `code`, `message` |

`index` là vị trí của event trong mảng request ban đầu, giúp device retry đúng item bị lỗi.

### SSE events

| Event | Khi nào gửi | Data |
| --- | --- | --- |
| `connected` | Ngay khi stream mở thành công | `client_id`, `timestamp` |
| `alert` | Khi device thuộc client gửi alert match filter | Alert payload realtime |
| `heartbeat` | Định kỳ mỗi 30 giây | `timestamp` |

---

## API đã triển khai

Base URL:

```text
http://localhost:8080/api/v1
```

| Method | Endpoint | Mục đích | Auth |
| --- | --- | --- | --- |
| POST | `/events` | Device gửi một alert event | Device API key |
| POST | `/events/batch` | Device gửi nhiều alert events trong một request | Device API key |
| GET | `/alerts/stream` | Client mở SSE stream nhận realtime alerts | Client JWT |
| GET | `/alerts/stream?device_id={id}` | Client mở SSE stream chỉ nhận alert của một device cụ thể | Client JWT |

---

## Chuẩn bị để test Backlog 2

Cần có:

1. `access_token` của client.
2. `device_id` của device thuộc client đó.
3. Raw `device_api_key` của device.

Có thể chuẩn bị bằng flow Backlog 1:

### 1. Login demo client

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

Lưu `data.access_token`.

### 2. Tạo device mới

```http
POST /api/v1/devices
Authorization: Bearer <access_token>
Content-Type: application/json
```

Request:

```json
{
  "name": "Warehouse Temperature Sensor",
  "type": "temperature_sensor",
  "status": "active",
  "tags": ["warehouse", "floor-1"],
  "metadata": {
    "location": "Room 101"
  }
}
```

Lưu:

```text
data.id      -> device_id
data.api_key -> device_api_key
```

Raw device API key chỉ hiển thị một lần, nên cần copy ngay khi tạo device hoặc rotate key.

---

## Luồng test chính cho reviewer

### 1. Mở realtime stream bằng client token

```http
GET /api/v1/alerts/stream
Authorization: Bearer <access_token>
Accept: text/event-stream
```

Response đầu tiên mong đợi:

```text
event: connected
data: {"client_id":"...","timestamp":"2026-05-03T12:00:00Z"}
```

Sau đó server sẽ giữ kết nối mở và gửi:

```text
event: alert
```

mỗi khi có device thuộc client hiện tại gửi alert.

Server cũng gửi heartbeat mỗi 30 giây:

```text
event: heartbeat
data: {"timestamp":"2026-05-03T12:00:30Z"}
```

---

### 2. Mở stream lọc theo device_id

```http
GET /api/v1/alerts/stream?device_id=<device_id>
Authorization: Bearer <access_token>
Accept: text/event-stream
```

Stream này chỉ nhận alert của đúng device được truyền trong query.

Nếu device khác cùng client gửi alert, stream có filter này không nhận event đó.

---

### 3. Device gửi một alert event

```http
POST /api/v1/events
Authorization: Bearer <device_api_key>
Content-Type: application/json
```

Request:

```json
{
  "type": "high_temperature",
  "severity": "warning",
  "message": "Temperature exceeded 80°C",
  "payload": {
    "temperature": 82.5,
    "unit": "celsius"
  },
  "occurred_at": "2026-05-03T12:00:00Z"
}
```

Response mong đợi: `202 Accepted`

```json
{
  "status": true,
  "message": "Event accepted",
  "data": {
    "alert_id": "9f3d2e1a-1234-4321-abcd-1234567890ab",
    "received_at": "2026-05-03T12:00:00.123Z"
  }
}
```

Nếu SSE stream đang mở, client sẽ thấy thêm event:

```text
event: alert
data: {"id":"9f3d2e1a-1234-4321-abcd-1234567890ab","device_id":"...","type":"high_temperature","severity":"warning","message":"Temperature exceeded 80°C",...}
```

---

### 4. Device gửi alert không có occurred_at

```http
POST /api/v1/events
Authorization: Bearer <device_api_key>
Content-Type: application/json
```

Request:

```json
{
  "type": "smoke_detected",
  "severity": "critical",
  "message": "Smoke detected in warehouse",
  "payload": {
    "zone": "A1"
  }
}
```

Response mong đợi: `202 Accepted`.

Nếu không gửi `occurred_at`, server dùng thời gian hiện tại làm thời điểm event xảy ra.

---

### 5. Device gửi batch alert events

```http
POST /api/v1/events/batch
Authorization: Bearer <device_api_key>
Content-Type: application/json
```

Request:

```json
{
  "events": [
    {
      "type": "temperature_recovered",
      "severity": "info",
      "message": "Temperature returned to normal",
      "payload": {
        "temperature": 24.1
      }
    },
    {
      "type": "smoke_detected",
      "severity": "critical",
      "message": "Smoke detected in warehouse",
      "payload": {
        "zone": "A1"
      }
    },
    {
      "type": "bad_event",
      "severity": "urgent",
      "message": "Invalid severity example"
    }
  ]
}
```

Response mong đợi: `202 Accepted`

```json
{
  "status": true,
  "message": "Batch processed",
  "data": {
    "accepted": 2,
    "rejected": 1,
    "alerts": [
      {
        "index": 0,
        "alert_id": "..."
      },
      {
        "index": 1,
        "alert_id": "..."
      }
    ],
    "errors": [
      {
        "index": 2,
        "code": "INVALID_SEVERITY",
        "message": "invalid alert severity"
      }
    ]
  }
}
```

Batch có thể vừa accepted vừa rejected để device biết chính xác item nào cần retry/sửa.

Giới hạn batch:

```text
Tối đa 100 events/request
```

---

## Business rules

- Device gửi alert bằng raw device API key qua header `Authorization: Bearer ah_dev_xxx`.
- Client nhận realtime alert bằng JWT access token qua `GET /alerts/stream`.
- Alert là append-only event; API hiện tại không update/delete alert.
- `type` là free string, không được rỗng và tối đa 100 ký tự.
- `message` không được rỗng.
- `severity` chỉ nhận `info`, `warning`, `critical`.
- `payload` là JSON metadata tùy chọn.
- `occurred_at` là thời điểm event xảy ra ở device; nếu không gửi thì server dùng thời gian hiện tại.
- `received_at` là thời điểm server nhận và lưu event.
- Mỗi alert gắn với đúng một `device_id` và một `client_id`.
- Device đã soft-delete không thể gửi event.
- Sau khi lưu alert, API phát PostgreSQL `NOTIFY`.
- SSE listener nhận event và fan-out tới subscriber phù hợp.
- SSE stream không gửi alert của client khác.
- Query `device_id` trên `/alerts/stream` chỉ nhận alert của đúng device đó.
- SSE gửi heartbeat mỗi 30 giây.
- Batch ingest nhận tối đa 100 events và trả lỗi theo từng index.

---

## Negative cases nên kiểm tra

| Case | Kết quả mong đợi |
| --- | --- |
| Gửi event không có device API key | `401 Unauthorized` |
| Gửi event bằng client JWT thay vì device API key | `401 Unauthorized` |
| Gửi event bằng device API key sai | `401 Unauthorized` |
| Device đã soft-delete gửi event | `401 Unauthorized` |
| Mở stream không có client JWT | `401 Unauthorized` |
| Mở stream bằng device API key thay vì client JWT | `401 Unauthorized` |
| `severity` không thuộc `info`, `warning`, `critical` | `400 Bad Request` hoặc item-level batch error |
| `type` rỗng | `400 Bad Request` hoặc item-level batch error |
| `message` rỗng | `400 Bad Request` hoặc item-level batch error |
| Batch rỗng | `400 Bad Request` |
| Batch hơn 100 events | `400 Bad Request` |
| Client A mở stream, device của Client B gửi alert | Stream của Client A không nhận alert |
| Stream có `device_id=A`, device B gửi alert | Stream không nhận alert của B |

---

## Kiểm tra last_seen_at sau khi gửi alert

Backlog 1 response device có thể có `last_seen_at`. Field này không lưu trực tiếp trong bảng `devices`; nó được tính từ alert mới nhất của device.

Trước khi device gửi alert:

```json
{
  "id": "...",
  "name": "Warehouse Temperature Sensor"
}
```

`last_seen_at` có thể không xuất hiện nếu chưa có alert.

Sau khi device gửi alert thành công:

```http
GET /api/v1/devices/{device_id}
Authorization: Bearer <access_token>
```

Response sẽ có `last_seen_at` tương ứng alert mới nhất:

```json
{
  "status": true,
  "message": "Device retrieved successfully",
  "data": {
    "id": "...",
    "name": "Warehouse Temperature Sensor",
    "last_seen_at": "2026-05-03T12:00:00Z"
  }
}
```

---

## Checklist reviewer cho Backlog 2

Có thể tick theo thứ tự này khi review:

- [ ] Chạy `make dev-up` thành công.
- [ ] Mở Swagger tại `http://localhost:8080/swagger/index.html`.
- [ ] Login demo client và lấy `access_token`.
- [ ] Tạo device mới và lưu `device_id`, `device_api_key`.
- [ ] Mở `GET /alerts/stream` bằng client JWT và thấy `event: connected`.
- [ ] Gửi một alert hợp lệ bằng `POST /events` với device API key.
- [ ] Xác nhận response single event là `202 Accepted` và có `alert_id`.
- [ ] Xác nhận SSE stream nhận được `event: alert`.
- [ ] Gọi `GET /devices/{device_id}` và xác nhận `last_seen_at` đã được cập nhật/tính ra.
- [ ] Mở stream với `?device_id=<device_id>` và xác nhận chỉ nhận alert của device đó.
- [ ] Gửi batch có cả event hợp lệ và event lỗi.
- [ ] Xác nhận batch response có `accepted`, `rejected`, `alerts`, `errors` đúng index.
- [ ] Gửi event không có device API key và xác nhận `401 Unauthorized`.
- [ ] Gửi event bằng client JWT thay vì device API key và xác nhận `401 Unauthorized`.
- [ ] Mở stream không có client JWT và xác nhận `401 Unauthorized`.
- [ ] Gửi severity không hợp lệ và xác nhận lỗi validation.
- [ ] Gửi batch rỗng hoặc hơn 100 events và xác nhận bị reject.
- [ ] Soft-delete device rồi thử gửi event bằng API key của device đó, xác nhận `401 Unauthorized`.

---

## Mapping yêu cầu đề bài sang implementation

| Yêu cầu đề bài | Implementation trong project |
| --- | --- |
| Thiết bị gửi event cảnh báo | `POST /api/v1/events` dùng device API key |
| Event được hệ thống tiếp nhận realtime | API lưu alert rồi phát PostgreSQL `NOTIFY` |
| Client nhận realtime alert | `GET /api/v1/alerts/stream` qua SSE |
| Device auth tách với client auth | Device dùng `ah_dev_xxx`, client dùng JWT access token |
| Hệ thống hỗ trợ nhiều event | `POST /api/v1/events/batch`, tối đa 100 events |
| Event lỗi không làm hỏng cả batch | Batch response trả `accepted/rejected` theo từng index |
| Alert gắn đúng client/device | Alert lưu cả `client_id` và `device_id` |
| Stream không lộ alert client khác | Fan-out filter theo `client_id` |

---

## Thiết kế realtime trong phạm vi challenge

Project dùng PostgreSQL `LISTEN/NOTIFY` và SSE thay vì Kafka/Redis vì phạm vi Backlog 2 chỉ cần realtime fan-out đơn giản trong local challenge.

```text
API instance
  ├── Insert alert vào PostgreSQL
  ├── Gửi pg_notify sau khi insert thành công
  ├── Listener nhận notification
  └── Stream service gửi event tới subscriber đang mở SSE
```

Cách này phù hợp vì:

- PostgreSQL đã là source of truth của project.
- Không cần thêm broker cho bài challenge hiện tại.
- SSE đơn giản hơn WebSocket cho luồng server push một chiều.
- Alert vẫn được lưu bền vững trước khi fan-out realtime.

Giới hạn hiện tại:

- Chưa có replay missed events qua stream; nếu client disconnect thì có thể xem lại qua future Backlog 3.
- Chưa có Redis/Kafka cho production scale nhiều instance lớn.
- Chưa có rate limiting/idempotency key cho device traffic rất lớn.

---

## Ghi chú cho reviewer

- `/events` và `/events/batch` không dùng client JWT; chúng dùng device API key.
- `/alerts/stream` không dùng device API key; nó dùng client JWT.
- Response ingest trả `202 Accepted` vì event đã được nhận/lưu và realtime fan-out là bước bất đồng bộ sau đó.
- Batch partial success là intentional: device có thể retry đúng item lỗi thay vì gửi lại toàn bộ batch.
- `payload` là JSON tự do, API không ép schema cụ thể để phù hợp nhiều loại device.
- `occurred_at` là thời điểm device ghi nhận event; `received_at` là thời điểm server nhận event.
- `last_seen_at` trên device được tính từ alert mới nhất, nên nó thể hiện hoạt động thực tế của device.

---

## Cách chạy local để test Backlog 2

Khởi động môi trường:

```bash
make dev-up
```

Mở Swagger:

```text
http://localhost:8080/swagger/index.html
```

Test SSE bằng curl:

```bash
curl -N \
  -H "Authorization: Bearer <access_token>" \
  -H "Accept: text/event-stream" \
  http://localhost:8080/api/v1/alerts/stream
```

Gửi event bằng curl:

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer <device_api_key>" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "high_temperature",
    "severity": "warning",
    "message": "Temperature exceeded 80°C",
    "payload": {"temperature": 82.5}
  }'
```

Chạy kiểm tra code:

```bash
go test ./...
go build ./...
docker compose config --quiet
```

Reset database nếu cần test từ đầu:

```bash
docker compose down -v
make dev-up
```
