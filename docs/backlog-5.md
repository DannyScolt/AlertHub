# Backlog 5 — Tìm kiếm alert

Backlog 5 mở rộng `GET /api/v1/alerts` bằng query param `search`, dùng chung response shape, pagination và filter của Backlog 3.

> Yêu cầu: Client có thể tìm alert theo nội dung message, type, tên device hoặc exact device UUID mà không làm lộ dữ liệu client khác.

---

## Mục tiêu đã triển khai

| Hạng mục | Trạng thái |
| --- | --- |
| Thêm `search` vào `GET /alerts`, không tạo endpoint mới | Hoàn thành |
| Trim search trước khi xử lý | Hoàn thành |
| Blank search được bỏ qua như không truyền search | Hoàn thành |
| Search non-empty phải dài 2..100 ký tự | Hoàn thành |
| Search case-insensitive trên `alerts.message`, `alerts.type`, `devices.name` | Hoàn thành |
| Nếu search là UUID thì match exact `alerts.device_id` | Hoàn thành |
| Search compose bằng AND với `device_id`, `severity`, `from`, `to`, `page`, `page_size` | Hoàn thành |
| Giữ response shape và ordering `occurred_at DESC, id DESC` | Hoàn thành |
| Giữ isolation bằng `alerts.client_id` và `devices.client_id` | Hoàn thành |
| Thêm `pg_trgm` + GIN trigram indexes cho reviewer-scale search | Hoàn thành |

---

## Query parameters

| Param | Bắt buộc | Ý nghĩa |
| --- | --- | --- |
| `search` | Không | Tìm trong alert message, alert type, device name, hoặc exact device UUID |
| `device_id` | Không | Lọc alert theo một device cụ thể |
| `severity` | Không | Lọc theo severity; có thể repeat nhiều lần |
| `from` | Không | RFC3339 inclusive lower bound của `occurred_at` |
| `to` | Không | RFC3339 inclusive upper bound của `occurred_at` |
| `page` | Không | Page bắt đầu từ 1 |
| `page_size` | Không | Số item mỗi page, tối đa 100 |

`search` được validate trong service:

| Input | Hành vi |
| --- | --- |
| Không truyền | Không áp search predicate |
| Chỉ whitespace | Trim thành blank và bỏ qua |
| 1 ký tự | `400 INVALID_SEARCH` |
| 2..100 ký tự | Hợp lệ |
| >100 ký tự | `400 INVALID_SEARCH` |

---

## Ví dụ request

Search theo message:

```bash
curl -s "http://localhost:8080/api/v1/alerts?search=smoke&page=1&page_size=20" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Search theo type:

```bash
curl -s "http://localhost:8080/api/v1/alerts?search=temperature" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Search theo tên device:

```bash
curl -s "http://localhost:8080/api/v1/alerts?search=Warehouse" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Search theo exact device UUID:

```bash
curl -s "http://localhost:8080/api/v1/alerts?search=$DEVICE_ID" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Search kết hợp filter Backlog 3:

```bash
curl -s "http://localhost:8080/api/v1/alerts?search=smoke&severity=critical&device_id=$DEVICE_ID&page_size=10" \
  -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Response vẫn là list alert cũ, không đổi shape:

```json
{
  "message": "Alerts retrieved successfully",
  "data": [
    {
      "id": "...",
      "device_id": "...",
      "type": "smoke_detected",
      "severity": "critical",
      "message": "Smoke detected in warehouse",
      "payload": {},
      "occurred_at": "2026-05-04T12:00:00Z",
      "received_at": "2026-05-04T12:00:00.123Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1,
    "has_next": false,
    "has_previous": false
  }
}
```

---

## Negative cases

| Case | Kỳ vọng |
| --- | --- |
| `GET /alerts?search=a` | `400 INVALID_SEARCH` |
| `GET /alerts?search=<101 ký tự>` | `400 INVALID_SEARCH` |
| `GET /alerts?search=%20%20` | `200 OK`, hành vi như không có search |
| Search không match alert nào | `200 OK`, `data: []`, pagination total bằng 0 |
| Client A search text chỉ có ở Client B | Không leak alert của Client B |

---

## Smoke test thủ công

Khởi động local và login:

```bash
docker compose up -d postgres redis api
ACCESS=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"client@example.com","password":"password123"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")
```

Tạo device có tên dễ search:

```bash
DEVICE_JSON=$(curl -s -X POST http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer $ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"name":"Warehouse Smoke Sensor","type":"smoke_detector","status":"active"}')
DEVICE_ID=$(python3 -c "import os,json;print(json.loads(os.environ['DEVICE_JSON'])['data']['id'])")
DEVICE_KEY=$(python3 -c "import os,json;print(json.loads(os.environ['DEVICE_JSON'])['data']['api_key'])")
```

Gửi alert:

```bash
curl -s -X POST http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer $DEVICE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"type":"smoke_detected","severity":"critical","message":"Smoke detected in warehouse","payload":{"zone":"A1"}}'
```

Kiểm tra search:

```bash
curl -s "http://localhost:8080/api/v1/alerts?search=smoke" -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
curl -s "http://localhost:8080/api/v1/alerts?search=smoke_detected" -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
curl -s "http://localhost:8080/api/v1/alerts?search=Warehouse" -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
curl -s "http://localhost:8080/api/v1/alerts?search=$DEVICE_ID" -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
curl -s "http://localhost:8080/api/v1/alerts?search=smoke&severity=critical&device_id=$DEVICE_ID" -H "Authorization: Bearer $ACCESS" | python3 -m json.tool
```

Đo latency p95 đơn giản:

```bash
for i in $(seq 1 20); do
  curl -s -o /dev/null -w "%{time_total}\n" \
    "http://localhost:8080/api/v1/alerts?search=smoke&page_size=20" \
    -H "Authorization: Bearer $ACCESS"
done | sort -n | tail -n 1
```

Kỳ vọng reviewer local p95 dưới 200ms sau khi migration index đã chạy.
