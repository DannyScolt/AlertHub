# Reviewer Test Flow — Backlog 1 đến Backlog 5

Tài liệu này hướng dẫn reviewer test nhanh toàn bộ flow chính của AlertHub theo thứ tự Backlog 1 → Backlog 5.

Base URL:

```text
http://localhost:8080/api/v1
```

Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

---

## 0. Chuẩn bị môi trường

Chạy project:

```bash
cp .env.example .env
make dev-up
```

Kiểm tra API sống:

```http
GET /health
```

Kỳ vọng:

```text
200 OK
```

Login demo client:

```http
POST /auth/login
Content-Type: application/json
```

Body:

```json
{
  "email": "client@example.com",
  "password": "password123"
}
```

Lưu token:

```text
data.access_token = ACCESS_TOKEN
```

Các API client dùng header:

```http
Authorization: Bearer <ACCESS_TOKEN>
```

---

## 1. Backlog 1 — Device registration và device list/filter

### 1.1 Tạo device

```http
POST /devices
Authorization: Bearer <ACCESS_TOKEN>
Content-Type: application/json
```

Body:

```json
{
  "name": "Warehouse Smoke Sensor",
  "type": "smoke_detector",
  "status": "active",
  "tags": ["warehouse", "smoke"],
  "metadata": {
    "zone": "A1"
  }
}
```

Kỳ vọng:

- `201 Created`
- Response có `data.id`
- Response có `data.api_key`
- Raw API key chỉ trả về một lần khi create/rotate

Lưu lại:

```text
data.id = DEVICE_ID
data.api_key = DEVICE_API_KEY
```

### 1.2 List devices

```http
GET /devices
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- `200 OK`
- Có device vừa tạo trong `data`
- Có pagination metadata

### 1.3 Filter devices theo status

```http
GET /devices?status=active
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Chỉ trả device có `status="active"`

### 1.4 Xem chi tiết device

```http
GET /devices/{DEVICE_ID}
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- `200 OK`
- Trả đúng device thuộc client hiện tại
- Không trả raw API key

### 1.5 Update device

```http
PATCH /devices/{DEVICE_ID}
Authorization: Bearer <ACCESS_TOKEN>
Content-Type: application/json
```

Body:

```json
{
  "name": "Warehouse Smoke Sensor Updated",
  "status": "maintenance",
  "tags": ["warehouse", "smoke", "updated"],
  "metadata": {
    "zone": "A1",
    "floor": 2
  }
}
```

Kỳ vọng:

- `200 OK`
- Device được cập nhật đúng

### 1.6 Rotate device API key

```http
POST /devices/{DEVICE_ID}/rotate-key
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- `200 OK`
- Response trả API key mới
- API key cũ không nên dùng tiếp cho ingest

Nếu test rotate, cập nhật lại:

```text
DEVICE_API_KEY = api_key mới
```

### 1.7 Soft delete và restore device

Soft delete:

```http
DELETE /devices/{DEVICE_ID}
Authorization: Bearer <ACCESS_TOKEN>
```

Restore:

```http
POST /devices/{DEVICE_ID}/restore
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Device bị soft-delete sẽ ẩn khỏi list mặc định
- Restore thành công thì device xuất hiện lại

---

## 2. Backlog 2 — Alert ingest, batch ingest và realtime SSE

Backlog 2 dùng device API key, không dùng client access token.

Header ingest:

```http
Authorization: Bearer <DEVICE_API_KEY>
```

### 2.1 Gửi một alert event

```http
POST /events
Authorization: Bearer <DEVICE_API_KEY>
Content-Type: application/json
```

Body:

```json
{
  "type": "smoke_detected",
  "severity": "critical",
  "message": "Smoke detected in warehouse",
  "payload": {
    "zone": "A1",
    "value": 95
  },
  "occurred_at": "2026-05-04T12:00:00Z"
}
```

Kỳ vọng:

- `201 Created`
- Response có alert `id`, `device_id`, `type`, `severity`, `message`, `payload`
- Không trả `client_id`
- Không trả API key/API key hash

### 2.2 Gửi batch events

```http
POST /events/batch
Authorization: Bearer <DEVICE_API_KEY>
Content-Type: application/json
```

Body:

```json
{
  "events": [
    {
      "type": "temperature_high",
      "severity": "warning",
      "message": "Temperature is high",
      "payload": {
        "value": 82
      }
    },
    {
      "type": "humidity_high",
      "severity": "info",
      "message": "Humidity is above normal",
      "payload": {
        "value": 70
      }
    }
  ]
}
```

Kỳ vọng:

- Batch hợp lệ trả accepted alerts
- Nếu có item lỗi, response thể hiện lỗi theo từng index
- Batch tối đa 100 events

### 2.3 Realtime SSE stream

Mở stream bằng client access token:

```bash
curl -N http://localhost:8080/api/v1/alerts/stream \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

Sau đó gửi lại `POST /events` bằng `DEVICE_API_KEY`.

Kỳ vọng:

- Stream nhận alert realtime
- Stream không nhận alert của client khác
- Có heartbeat định kỳ

---

## 3. Backlog 3 — Alert query/filter/pagination

Backlog 3 dùng client access token.

### 3.1 List alerts mặc định

```http
GET /alerts
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- `200 OK`
- `data` là array alerts
- Có `pagination`
- Không có `client_id` trong từng alert item
- Order mới nhất trước: `occurred_at DESC, id DESC`

### 3.2 Pagination

```http
GET /alerts?page=1&page_size=20
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- `pagination.page = 1`
- `pagination.page_size = 20`
- Có `total`, `total_pages`, `has_next`, `has_previous`

### 3.3 Filter theo device

```http
GET /alerts?device_id={DEVICE_ID}
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Chỉ trả alert của device đó

### 3.4 Filter theo severity

```http
GET /alerts?severity=critical
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Chỉ trả alert `severity="critical"`

### 3.5 Filter nhiều severity

```http
GET /alerts?severity=warning&severity=critical
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Chỉ trả alert thuộc `warning` hoặc `critical`

### 3.6 Filter theo time range

```http
GET /alerts?from=2026-05-04T00:00:00Z&to=2026-05-04T23:59:59Z
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Chỉ trả alert có `occurred_at` trong khoảng

### 3.7 Negative cases

Invalid device ID:

```http
GET /alerts?device_id=abc
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

```text
400 INVALID_DEVICE_ID
```

Invalid severity:

```http
GET /alerts?severity=urgent
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

```text
400 INVALID_SEVERITY
```

Invalid pagination:

```http
GET /alerts?page=0
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

```text
400 INVALID_PAGINATION
```

---

## 4. Backlog 4 — Auto-escalation critical alert

Backlog 4 tự tạo alert critical khi một device gửi nhiều event cùng `type` trong rolling window.

Default config development:

```text
ESCALATION_THRESHOLD=3
ESCALATION_WINDOW=60s
ESCALATION_COOLDOWN=5m
```

### 4.1 Gửi 3 events cùng type

Gửi request này 3 lần trong vòng 60 giây:

```http
POST /events
Authorization: Bearer <DEVICE_API_KEY>
Content-Type: application/json
```

Body:

```json
{
  "type": "smoke_burst",
  "severity": "warning",
  "message": "Smoke burst warning",
  "payload": {
    "zone": "A1"
  }
}
```

### 4.2 Query critical alerts

```http
GET /alerts?severity=critical&page_size=20
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng thấy alert tự sinh:

```json
{
  "type": "auto_escalated",
  "severity": "critical",
  "message": "Repeated alert burst auto-escalated",
  "payload": {
    "source_alert_ids": ["...", "...", "..."],
    "count": 3,
    "threshold": 3,
    "window_seconds": 60
  }
}
```

### 4.3 Test cooldown

Gửi tiếp 3 events cùng `type="smoke_burst"` ngay lập tức.

Sau đó query lại:

```http
GET /alerts?severity=critical&page_size=20
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Không sinh duplicate `auto_escalated` ngay trong cooldown window

### 4.4 Search auto-escalated alert

```http
GET /alerts?search=auto_escalated&severity=critical
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Có thể tìm alert auto-escalated qua Backlog 5 search

---

## 5. Backlog 5 — Alert search

Backlog 5 mở rộng `GET /alerts` bằng query param `search`.

Search có thể match:

- `alerts.message`
- `alerts.type`
- `devices.name`
- exact `alerts.device_id` nếu search là UUID

### 5.1 Search theo message

```http
GET /alerts?search=smoke
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Trả alert có message chứa `smoke` không phân biệt hoa thường

### 5.2 Search theo alert type

```http
GET /alerts?search=smoke_detected
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Trả alert có `type="smoke_detected"`

### 5.3 Search theo device name

```http
GET /alerts?search=Warehouse
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Trả alert của device có name chứa `Warehouse`

### 5.4 Search theo exact device UUID

```http
GET /alerts?search={DEVICE_ID}
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Trả alert của đúng `DEVICE_ID`

### 5.5 Search kết hợp filter Backlog 3

```http
GET /alerts?search=smoke&severity=critical&device_id={DEVICE_ID}&page=1&page_size=10
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- Search compose bằng AND với `severity`, `device_id`, pagination
- Response shape không đổi

### 5.6 Blank search

```http
GET /alerts?search=%20%20
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

- `200 OK`
- Hành vi giống như không truyền `search`

### 5.7 Invalid search quá ngắn

```http
GET /alerts?search=a
Authorization: Bearer <ACCESS_TOKEN>
```

Kỳ vọng:

```text
400 INVALID_SEARCH
```

### 5.8 Cross-client isolation

Tạo client thứ hai, tạo device và alert với text riêng cho client thứ hai.

Sau đó dùng token client thứ nhất gọi:

```http
GET /alerts?search=<text chỉ có ở client thứ hai>
Authorization: Bearer <ACCESS_TOKEN_CLIENT_1>
```

Kỳ vọng:

- `200 OK`
- `data: []`
- Không leak alert của client khác

---

## 6. Checklist kết quả cuối

Reviewer có thể xem là đạt nếu:

- Backlog 1 tạo/list/filter/update/delete/restore device chạy đúng
- Backlog 2 ingest single/batch alert chạy đúng
- SSE stream nhận realtime alert
- Backlog 3 list/filter/pagination alert đúng và không leak `client_id`
- Backlog 4 tự sinh `auto_escalated` critical alert khi đủ threshold
- Backlog 4 cooldown không sinh duplicate ngay
- Backlog 5 search theo message/type/device name/device UUID chạy đúng
- Search compose với filter Backlog 3
- Cross-client isolation luôn đúng

Chạy verification tổng:

```bash
go test ./...
go build ./...
go vet ./...
docker compose config --quiet
```
