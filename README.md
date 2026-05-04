# AlertHub API

AlertHub là REST API viết bằng Go cho bài **Backend Coding Challenge**. Bài nộp này đã triển khai Backlog 1, Backlog 2, Backlog 3, Backlog 4 và Backlog 5:

> Backlog 1: Là một client, tôi có thể đăng ký thiết bị mới vào hệ thống và truy vấn danh sách thiết bị theo trạng thái.
>
> Backlog 2: Thiết bị gửi event cảnh báo realtime lên hệ thống.
>
> Backlog 3: Client xem/lọc danh sách cảnh báo theo thiết bị, mức độ nghiêm trọng và thời gian.
>
> Backlog 4: Hệ thống tự nâng cảnh báo lên critical khi một device gửi nhiều event cùng loại trong rolling window.
>
> Backlog 5: Client tìm kiếm cảnh báo theo nội dung alert, loại alert, tên device hoặc exact device UUID.

Project có đầy đủ authentication cho client, device API key auth, alert ingestion, realtime SSE stream, alert query API với filter/search/pagination, auto-escalation qua PostgreSQL LISTEN/NOTIFY + Redis cooldown, PostgreSQL storage, Swagger documentation, Docker local development và cấu trúc code theo từng layer để reviewer có thể chạy/test nhanh.

---

## Phạm Vi Backlog

### Đã triển khai

| Backlog | Yêu cầu | Trạng thái |
| --- | --- | --- |
| 1 | Client có thể đăng ký thiết bị mới và xem danh sách thiết bị theo trạng thái | Hoàn thành |
| 2 | Thiết bị gửi event cảnh báo realtime lên hệ thống | Hoàn thành |
| 3 | Client xem/lọc danh sách cảnh báo theo thiết bị, mức độ nghiêm trọng và thời gian | Hoàn thành |
| 4 | Hệ thống tự nâng cảnh báo lên critical khi có nhiều event cùng loại trong 60 giây | Hoàn thành |
| 5 | Client tìm kiếm alert theo message, type, tên device hoặc exact device UUID | Hoàn thành |

---

## Thời Gian Thực Hiện

| Phần | Thời gian ước tính |
| --- | --- |
| Backlog 1 — Auth, device registration, device list/filter | 8 giờ |
| Backlog 2 — Alert ingest, batch ingest, SSE realtime stream | 8 giờ |
| Backlog 3 — Alert query/filter/pagination và chuẩn hóa query structure | 6 giờ |
| Backlog 4 — Auto-escalation, Redis cooldown, listener và smoke tests | 6 giờ |
| Backlog 5 — Alert search, indexes, docs và Swagger | 4 giờ |
| Refactor theo SOLID, verification, reviewer docs | 6 giờ |

Tổng thời gian ước tính: khoảng 38 giờ.

---

## Quick Start Cho Reviewer

```bash
cp .env.example .env
make dev-up
```

Sau khi API start:

```text
API:     http://localhost:8080
Swagger: http://localhost:8080/swagger/index.html
Adminer: http://localhost:8081
```

Demo client development:

```text
Email: client@example.com
Password: password123
```

Kiểm tra nhanh trước khi review:

```bash
go test ./...
go build ./...
go vet ./...
docker compose config --quiet
```

---

## Công Nghệ Sử Dụng

| Hạng mục | Công nghệ |
| --- | --- |
| Ngôn ngữ | Go |
| HTTP framework | Gin |
| Database | PostgreSQL |
| Cooldown store | Redis có password |
| Database driver | pgx / pgxpool |
| Authentication | JWT access token + opaque refresh token |
| Hash password | bcrypt |
| API docs | Swagger / OpenAPI qua swaggo |
| Local runtime | Docker Compose |
| Database inspection | Adminer |
| Hot reload | Air |

---

## Kiến Trúc Project

```text
cmd/api                  # Điểm khởi chạy API
core/config              # Load và quản lý environment config
core/database            # Kết nối PostgreSQL
core/domain              # Domain models và enums
core/dto                 # HTTP request/response DTOs
core/repository          # PostgreSQL repositories
core/services            # Business logic
core/handlers            # Gin HTTP handlers
core/router              # Đăng ký routes và wiring dependencies
core/middleware          # Global middleware và auth middleware
core/server              # Bootstrap Gin server
core/utils               # Shared helpers
migrations               # PostgreSQL migrations
docs                     # Swagger docs generated
```

Luồng xử lý chính:

```text
router -> handler -> service -> repository -> PostgreSQL
```

`core/server` chỉ bootstrap Gin và gọi router tổng. Việc wire repository/service/handler nằm trong từng module router để giữ cấu trúc rõ ràng và dễ theo dõi.

---

## Quyết Định Thiết Kế Quan Trọng

| Quyết định | Lý do |
| --- | --- |
| Dùng PostgreSQL làm storage chính | Phù hợp dữ liệu quan hệ `clients`/`devices`/`alerts`, hỗ trợ transaction, index, enum và JSONB payload |
| Lưu refresh token dạng hash trong `client_tokens` | Không lưu raw refresh token, hỗ trợ rotate/revoke session an toàn hơn |
| Device API key chỉ trả một lần và chỉ lưu hash | Giảm rủi ro lộ secret nếu database bị đọc trực tiếp |
| Alert là append-only event | Phù hợp audit/history; không update/delete alert đã nhận |
| SSE cho realtime stream | Đủ cho one-way realtime alert từ server về client, đơn giản hơn WebSocket cho scope challenge |
| PostgreSQL `LISTEN/NOTIFY` cho alert fan-out nội bộ | Tái dùng Postgres, tránh thêm message broker khi chưa cần Kafka/RabbitMQ |
| Redis cho auto-escalation cooldown | `SET NX EX` giúp claim cooldown atomic, tránh emit duplicate critical alert |
| Backlog 5 search nằm trong `GET /alerts` | Search là filter của alert history nên dùng chung pagination/order/isolation của Backlog 3 |
| `ILIKE` + `pg_trgm` cho search | Reviewer cần substring search theo message/type/device name; trigram index giúp giữ latency thấp |
| Tách handler/service/repository theo focused interfaces | Giữ code dễ test, tránh service phụ thuộc repository methods không liên quan |

---

## Cấu Trúc Database

```text
clients
└── Lưu thông tin client đăng ký hệ thống. Có remember_token nullable cho quen thuộc, nhưng refresh token không lưu ở đây.

client_tokens
└── Lưu auth session của client, refresh-token hash, token rotation chain, trạng thái revoke/logout, user agent và IP metadata.

devices
└── Lưu thiết bị IoT thuộc về client, API key hash, type/status enum, tags, metadata, timestamps và soft-delete state.

alerts
└── Lưu event cảnh báo append-only do device gửi lên, gồm device_id, client_id, type, severity, message, payload và timestamps.
```

Các quyết định bảo mật quan trọng:

- Password được lưu bằng bcrypt hash.
- Refresh token chỉ lưu dạng hash trong bảng `client_tokens`.
- Device API key chỉ lưu dạng hash.
- Raw device API key chỉ trả về một lần khi tạo device hoặc rotate key.
- Auth response chỉ trả token, không trả thông tin profile của client.
- Alert event không chứa hoặc trả về `api_key` / `api_key_hash`.
- SSE stream dùng JWT của client, còn ingest event dùng device API key.

---

## Yêu Cầu Môi Trường

- Go 1.25+
- Docker và Docker Compose
- Make

Nếu muốn chạy local không dùng Docker thì cần thêm:

- PostgreSQL 16+

---

## Cấu Hình Environment

Copy file environment mẫu:

```bash
cp .env.example .env
```

Nếu chạy API trực tiếp trên host machine, `DATABASE_URL` nên dùng `localhost`:

```env
DATABASE_URL=postgres://alerthub:alerthub@localhost:5432/alerthub?sslmode=disable
```

Khi chạy bằng Docker Compose, API container sẽ override `DATABASE_URL` để trỏ tới Docker service name là `postgres`.

Backlog 4 dùng Redis cho cooldown atomic khi auto-escalate. Khi burst đạt threshold, API append thêm alert `severity="critical"`, `type="auto_escalated"`; payload có `source_alert_ids` là đầy đủ alert IDs nguồn trong rolling window. API tự build Redis connection URL nội bộ từ các biến dưới đây:

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

Khi chạy bằng Docker Compose, API container override `REDIS_HOST=redis`. Ở `APP_ENV=staging` hoặc `APP_ENV=production`, `REDIS_PASSWORD` không được rỗng và không được dùng giá trị development mặc định.

---

## Chạy Project Bằng Docker

Cách chạy tối giản nhất:

```bash
cp .env.example .env
make dev-up
```

Lệnh `make dev-up` sẽ khởi động PostgreSQL, Adminer, chạy migrations và start API.

Lệnh này sẽ chạy:

```text
1. docker compose up -d postgres adminer
2. docker compose run --rm migrate
3. docker compose up -d api
```

API sẽ chạy tại:

```text
http://localhost:8080
```

Dừng containers:

```bash
make docker-down
```

Xem logs:

```bash
make docker-logs
```

Reset database local trong Docker:

```bash
docker compose down -v
make dev-up
```

---

## Swagger

Mở Swagger UI tại:

```text
http://localhost:8080/swagger/index.html
```

Regenerate Swagger docs sau khi chỉnh handlers hoặc DTOs:

```bash
make swagger
```

---

## Adminer

Mở Adminer tại:

```text
http://localhost:8081
```

Thông tin đăng nhập:

```text
System: PostgreSQL
Server: postgres
Username: alerthub
Password: alerthub
Database: alerthub
```

Trong Adminer phải dùng `postgres` ở ô Server. Không dùng `localhost`, vì Adminer đang chạy bên trong Docker container riêng.

---

## Demo Client

Ở môi trường development, API tự seed sẵn một client để reviewer test nhanh bằng Swagger:

```text
Email: client@example.com
Password: password123
```

Seed này sẽ bị skip khi `APP_ENV=production`.

---

## Luồng Test Backlog 1 Cho Reviewer

Chi tiết Backlog 1 đã được tách riêng để dễ đọc và test theo từng bước:

- [docs/backlog-1.md](docs/backlog-1.md) — client đăng nhập, đăng ký thiết bị, xem danh sách thiết bị và lọc theo trạng thái.

---

## Luồng Test Backlog 2 Cho Reviewer

Chi tiết Backlog 2 đã được tách riêng để dễ đọc và test theo từng bước:

- [docs/backlog-2.md](docs/backlog-2.md) — device gửi alert event, batch ingest, realtime SSE stream và các negative cases.

---

## Luồng Test Backlog 3 Cho Reviewer

Chi tiết Backlog 3 đã được tách riêng để dễ đọc và test theo từng bước:

- [docs/backlog-3.md](docs/backlog-3.md) — client xem và lọc danh sách alert theo device, severity và thời gian, có pagination.

---

## Luồng Test Backlog 4 Cho Reviewer

Chi tiết Backlog 4 đã được tách riêng để dễ đọc và test theo từng bước:

- [docs/backlog-4.md](docs/backlog-4.md) — auto-escalation critical alert, Redis cooldown, payload `source_alert_ids` đầy đủ, smoke test burst/cooldown/cross-client/latency.

---

## Luồng Test Backlog 5 Cho Reviewer

Chi tiết Backlog 5 đã được tách riêng để dễ đọc và test theo từng bước:

- [docs/backlog-5.md](docs/backlog-5.md) — tìm kiếm alert theo message/type/tên device/exact device UUID, compose với filter Backlog 3 và giữ nguyên response shape.

---

## API Reference

Base URL:

```text
http://localhost:8080/api/v1
```

### Health

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| GET | `/health` | Kiểm tra API có đang chạy không |

### Auth

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| POST | `/auth/register` | Đăng ký client mới và cấp token |
| POST | `/auth/login` | Đăng nhập và cấp token |
| POST | `/auth/refresh` | Rotate refresh token và cấp token mới |
| POST | `/auth/logout` | Logout bằng access token của client hiện tại |
| POST | `/auth/logout-all` | Logout toàn bộ session của client hiện tại |
| GET | `/auth/sessions` | Xem danh sách session của client hiện tại |
| DELETE | `/auth/sessions/{id}` | Revoke một session bằng session ID |

### Client

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| GET | `/clients/me` | Xem profile của client đang đăng nhập |

### Devices

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| POST | `/devices` | Đăng ký thiết bị mới |
| GET | `/devices` | Xem danh sách thiết bị, có filter và pagination |
| GET | `/devices/{id}` | Xem chi tiết một thiết bị |
| PATCH | `/devices/{id}` | Cập nhật một thiết bị |
| DELETE | `/devices/{id}` | Soft delete một thiết bị |
| POST | `/devices/{id}/restore` | Khôi phục thiết bị đã soft-delete |
| POST | `/devices/{id}/rotate-key` | Rotate device API key |

### Alerts / Events

| Method | Endpoint | Mô tả |
| --- | --- | --- |
| POST | `/events` | Device gửi một alert event realtime |
| POST | `/events/batch` | Device gửi nhiều alert events trong một request, tối đa 100 events |
| GET | `/alerts` | Client xem/lọc/tìm kiếm danh sách alert theo device, severity, thời gian, search, có pagination |
| GET | `/alerts/stream` | Client mở SSE stream để nhận realtime alerts |

---

## Quy Tắc Device

### Device Status

```text
active
inactive
maintenance
error
```

### Device Type

```text
temperature_sensor
humidity_sensor
smoke_detector
motion_sensor
door_sensor
camera
gateway
other
```

### Business Rules

- Một device thuộc về đúng một client.
- Tên device phải unique theo từng client trong nhóm device chưa bị delete.
- Device type và status được validate bằng enum cố định.
- Delete device là soft delete bằng field `deleted_at`.
- Soft-deleted devices bị ẩn mặc định khỏi list/detail API.
- Device đã delete có thể restore trong restore window được cấu hình.
- Device sau khi restore sẽ có status là `inactive`.
- Device đã delete không thể update hoặc rotate API key.
- Device API key chỉ được trả về khi create hoặc rotate.
- Hệ thống chỉ lưu hash của device API key.
- `last_seen_at` trong response device được tính từ alert mới nhất của device, không lưu trực tiếp trong bảng `devices`.

---

## Quy Tắc Alert

### Alert Severity

```text
info
warning
critical
```

### Business Rules

- Device gửi alert bằng raw device API key qua header `Authorization: Bearer ah_dev_xxx`.
- Client nhận realtime alert bằng JWT access token qua `GET /alerts/stream`.
- Alert là append-only event; API hiện tại không update/delete alert.
- `type` là free string, không được rỗng và tối đa 100 ký tự.
- `message` không được rỗng.
- `payload` là JSON metadata tùy chọn.
- `occurred_at` là thời điểm event xảy ra ở device; nếu không gửi thì server dùng thời gian hiện tại.
- `received_at` là thời điểm server nhận và lưu event.
- Mỗi alert gắn với đúng một `device_id` và một `client_id`.
- Sau khi lưu alert, API phát PostgreSQL `NOTIFY`; SSE listener nhận event và fan-out tới subscriber phù hợp.
- SSE stream không gửi alert của client khác.
- Query `device_id` trên `/alerts/stream` chỉ nhận alert của đúng device đó.
- SSE gửi heartbeat mỗi 30 giây.
- Batch ingest nhận tối đa 100 events và trả lỗi theo từng index.
- `GET /alerts?search=...` tìm case-insensitive trên alert message, alert type, device name và exact device UUID nếu search là UUID.
- Search compose bằng AND với filter Backlog 3 (`device_id`, `severity`, `from`, `to`) và không đổi response shape.

---

## Lệnh Hữu Ích

| Lệnh | Mô tả |
| --- | --- |
| `make dev-up` | Start Postgres, Adminer, migrations và API |
| `make docker-up` | Start Docker services |
| `make docker-down` | Stop Docker services |
| `make docker-logs` | Xem API logs |
| `make migrate-up` | Chạy migrations qua Docker |
| `make migrate-down` | Rollback migrations qua Docker |
| `make swagger` | Regenerate Swagger docs |
| `make test` | Chạy `go test ./...` |
| `make build` | Chạy `go build ./...` |
| `make tidy` | Chạy `go mod tidy` |

---

## Verification

Trước khi submit hoặc sau khi chỉnh code, chạy:

```bash
go test ./...
go build ./...
docker compose config --quiet
```

Kiểm tra fresh database schema:

```bash
docker compose down -v
make dev-up
docker compose exec -T postgres psql -U alerthub -d alerthub -c "SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name;"
```

Các bảng mong đợi:

```text
alerts
client_tokens
clients
devices
schema_migrations
```

---

## Giới Hạn Hiện Tại

Project này cố ý triển khai Backlog 1 đến Backlog 5.

Chưa bao gồm trong phạm vi hiện tại:

- Rate limiting, idempotency key, partitioning bảng alert cho production traffic rất lớn.
- Production Docker image hardening.
- CI/CD pipeline.

Nếu tiếp tục phát triển production-scale, hướng tiếp cận sẽ là:

| Phần còn lại | Cách tiếp cận |
| --- | --- |
| Rate limiting | Thêm middleware theo client/device key, dùng Redis counter theo window |
| Idempotency key | Lưu idempotency key theo client/device + request hash để chống ingest trùng |
| Partition alert table | Partition `alerts` theo thời gian hoặc client khi dữ liệu tăng lớn |
| Message broker | Chuyển LISTEN/NOTIFY sang Kafka/RabbitMQ nếu cần fan-out nhiều worker hoặc retry phức tạp |
| Production hardening | Multi-stage Dockerfile, non-root user, healthcheck, resource limits, secret management |
| CI/CD | Pipeline chạy test/build/vet/swagger check và migration validation trước merge/deploy |

---

## Ghi Chú Nộp Bài

- Có thể nộp bằng GitHub repository public/private; nếu private thì cấp quyền truy cập cho reviewer.
- Có thể nộp bằng file ZIP nếu reviewer yêu cầu qua email.
- API đã có Swagger và có thể test trực tiếp bằng Swagger UI.
- Development environment có seed demo client để reviewer test nhanh.
- Database schema dùng `clients`, `client_tokens`, `devices`, và `alerts` để giữ mental model rõ ràng cho Backlog 1 đến Backlog 5.
- `client_tokens` lưu auth session/refresh-token metadata; raw refresh token không bao giờ được lưu trực tiếp.
- `alerts` lưu realtime event append-only; realtime delivery dùng PostgreSQL LISTEN/NOTIFY và SSE.
- Auto-escalation tạo alert mới `severity="critical"`, `type="auto_escalated"` khi cùng `(device_id,type)` đạt ngưỡng trong window; Redis đảm bảo cooldown atomic.
- Device ingest dùng device API key, còn client stream dùng JWT access token.
