# Backlog 4 — Tự động nâng cảnh báo lên critical

Backlog 4 tự phát hiện burst alert cùng loại của cùng một device và append thêm một alert critical mới để client thấy qua API/query/stream hiện có.

> Yêu cầu: Khi một device tạo nhiều event cùng `type` trong rolling window, hệ thống tự emit alert `severity="critical"`, `type="auto_escalated"` nếu chưa nằm trong cooldown.

---

## Mục tiêu đã triển khai

| Hạng mục | Trạng thái |
| --- | --- |
| Detection chạy bất đồng bộ qua PostgreSQL LISTEN/NOTIFY | Hoàn thành |
| Không thêm logic blocking vào ingest path | Hoàn thành |
| Đếm cùng `(device_id, type)` trong rolling window | Hoàn thành |
| Emit alert mới append-only với `type="auto_escalated"` | Hoàn thành |
| Cooldown atomic bằng Redis `SET NX EX` | Hoàn thành |
| Redis có password và fail-fast ở staging/production nếu dùng password mặc định/rỗng | Hoàn thành |
| Auto-escalated alert xuất hiện qua `GET /alerts` và `/alerts/stream` | Hoàn thành |
| Service/listener theo pattern Backlog 3: focused interfaces, Input/Outcome, stub tests | Hoàn thành |

---

## Cấu hình

```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=change-me-redis-password
REDIS_DB=0
ESCALATION_ENABLED=true
ESCALATION_THRESHOLD=3
ESCALATION_WINDOW=60s
ESCALATION_COOLDOWN=5m
```

API tự build Redis connection URL nội bộ từ `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`; `.env` không cần khai báo thêm URL trùng lặp.

| Biến | Default | Ý nghĩa |
| --- | --- | --- |
| `REDIS_HOST` | `localhost` | Host Redis khi chạy API trực tiếp trên host machine |
| `REDIS_PORT` | `6379` | Port Redis |
| `REDIS_PASSWORD` | `change-me-redis-password` | Password Redis |
| `REDIS_DB` | `0` | Redis database index |
| `ESCALATION_ENABLED` | `true` | Tắt/bật service escalation |
| `ESCALATION_THRESHOLD` | `3` | Số event cùng loại cần đạt để escalate |
| `ESCALATION_WINDOW` | `60s` | Rolling window để đếm burst |
| `ESCALATION_COOLDOWN` | `5m` | Thời gian chặn emit lặp cho cùng `(device_id,type)` |

Ở `APP_ENV=staging` hoặc `APP_ENV=production`, API từ chối start nếu Redis password rỗng hoặc bằng `change-me-redis-password`.

---

## Luồng xử lý

```text
Device
  │ POST /api/v1/events
  ▼
Alert ingest service
  │ insert alert thường
  │ pg_notify alert_channel
  ▼
EscalationListener
  │ nhận notification async
  │ gọi EscalationService.HandleNewAlert
  ▼
EscalationService
  │ lookup alert gốc
  │ skip nếu type="auto_escalated"
  │ count cùng device/type trong ESCALATION_WINDOW
  │ claim Redis cooldown bằng SET NX EX
  ▼
Insert alert critical mới
  │ type="auto_escalated"
  │ pg_notify tiếp để query/stream thấy như alert thường
  ▼
Client GET /alerts hoặc /alerts/stream
```

---

## Payload của alert auto-escalated

Alert mới có:

| Field | Giá trị |
| --- | --- |
| `severity` | `critical` |
| `type` | `auto_escalated` |
| `device_id` | Giống alert gốc |
| `client_id` | Giống alert gốc |
| `payload.source_alert_ids` | Danh sách đầy đủ alert ID nguồn cùng `(device_id,type)` nằm trong rolling window, sắp xếp theo `occurred_at ASC, id ASC` |
| `payload.count` | Số event cùng type trong window, bằng độ dài `source_alert_ids` |
| `payload.window_seconds` | Window tính bằng giây |
| `payload.threshold` | Threshold cấu hình |
| `payload.detected_at` | Thời điểm service phát hiện burst |

Ví dụ item trong `GET /api/v1/alerts`:

```json
{
  "id": "...",
  "device_id": "...",
  "type": "auto_escalated",
  "severity": "critical",
  "message": "Repeated alert burst auto-escalated",
  "payload": {
    "source_alert_ids": [
      "0981a597-640a-4090-b0ee-49c9a5e9ed83",
      "73b7126d-c585-4ee3-a3cf-f5500fcc40b5",
      "33b537f3-0439-4a55-85b1-7f64da5cb1a8"
    ],
    "count": 3,
    "window_seconds": 60,
    "threshold": 3,
    "detected_at": "2026-05-04T12:00:00Z"
  },
  "occurred_at": "2026-05-04T12:00:00Z",
  "received_at": "2026-05-04T12:00:00.123Z"
}
```

---

## Edge cases

| Case | Hành vi |
| --- | --- |
| `ESCALATION_ENABLED=false` | Service return ngay, không lookup/count/cooldown/insert |
| Chưa đủ threshold | Không emit critical |
| Đang trong cooldown | Không emit thêm critical |
| Cooldown hết và burst vẫn đủ threshold | Có thể emit critical mới |
| Redis down sau khi API đã chạy | Service log lỗi và không crash API |
| Nhận notification của `auto_escalated` | Service skip để tránh recursion |
| Client khác query/stream | Không thấy alert của client không thuộc mình |
| Kiểm tra log sau smoke | Redis password không xuất hiện plain text; log chỉ hiển thị URL đã mask password |

---

## Checklist từng yêu cầu cho reviewer

| Yêu cầu | Cách kiểm tra nhanh |
| --- | --- |
| Burst đủ threshold emit critical | Gửi 3 event cùng `type`, query `/alerts?severity=critical`, thấy đúng 1 item `type="auto_escalated"` |
| Payload đủ source IDs | `payload.source_alert_ids` có đủ 3 alert IDs nguồn và `payload.count=3` |
| Cooldown không emit lặp | Gửi tiếp 3 event cùng `type` ngay sau burst đầu, số `auto_escalated` không tăng |
| Below threshold không emit | Gửi 1-2 event type khác trong window, không có critical mới |
| Cross-client isolation | Login/tạo client khác, query `/alerts`, không thấy escalation của client cũ |
| Async/latency | Đo latency ingest; p95 phải dưới 200ms |
| Redis auth | `docker compose exec -T redis redis-cli -a <REDIS_PASSWORD> ping` trả `PONG` |
| Fail-fast production | Start với `APP_ENV=production` và password default/rỗng phải exit non-zero |
| Không lộ secret | `docker compose logs api` không chứa plain Redis password |

---

## Smoke test thủ công

Khởi động local:

```bash
docker compose up -d redis postgres api
```

Login và chuẩn bị device giống Backlog 1/2:

```bash
ACCESS=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"client@example.com","password":"password123"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")
```

Tạo device rồi lưu raw API key:

```bash
DEVICE_JSON=$(curl -s -X POST http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer $ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"name":"Escalation Sensor","type":"temperature_sensor","status":"active"}')
DEVICE_KEY=$(python3 -c "import os,json;print(json.loads(os.environ['DEVICE_JSON'])['data']['api_key'])")
```

Gửi 3 event cùng `type` trong vài giây:

```bash
for i in 1 2 3; do
  curl -s -X POST http://localhost:8080/api/v1/events \
    -H "Authorization: Bearer $DEVICE_KEY" \
    -H "Content-Type: application/json" \
    -d '{"type":"high_temperature","severity":"warning","message":"Temperature burst","payload":{"value":82}}'
done
```

Kiểm tra alert auto-escalated:

```bash
curl -s "http://localhost:8080/api/v1/alerts?severity=critical&page_size=20" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Kỳ vọng có đúng một item `type="auto_escalated"` cho burst đầu tiên. Trong item đó:

- `payload.source_alert_ids` có đủ 3 `alert_id` nguồn trong rolling window.
- `payload.count=3`.
- `payload.window_seconds=60`.
- `payload.threshold=3`.

Gửi tiếp 3 event cùng loại ngay lập tức thì không sinh thêm critical mới vì cooldown còn active.

Kiểm tra Redis auth và log masking:

```bash
docker compose exec -T redis redis-cli -a "$REDIS_PASSWORD" ping
docker compose logs --tail=300 api | grep -F "$REDIS_PASSWORD"
```

Lệnh đầu kỳ vọng `PONG`. Lệnh grep log kỳ vọng không có output, vì API chỉ log Redis URL đã mask password.

---

## Verification code

```bash
go test ./core/repository/escalation
go test ./core/services/alert
go test ./core/infra/redis
go test ./...
go build ./...
go vet ./...
docker compose config --quiet
```
