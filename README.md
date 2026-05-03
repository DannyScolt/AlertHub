# AlertHub API

AlertHub là REST API viết bằng Go cho bài **Backend Coding Challenge**. Bài nộp này đã triển khai Backlog 1 và Backlog 2:

> Backlog 1: Là một client, tôi có thể đăng ký thiết bị mới vào hệ thống và truy vấn danh sách thiết bị theo trạng thái.
>
> Backlog 2: Thiết bị gửi event cảnh báo realtime lên hệ thống.

Project có đầy đủ authentication cho client, device API key auth, alert ingestion, realtime SSE stream, PostgreSQL storage, Swagger documentation, Docker local development và cấu trúc code theo từng layer để reviewer có thể chạy/test nhanh.

---

## Phạm Vi Backlog

### Đã triển khai

| Backlog | Yêu cầu | Trạng thái |
| --- | --- | --- |
| 1 | Client có thể đăng ký thiết bị mới và xem danh sách thiết bị theo trạng thái | Hoàn thành |
| 2 | Thiết bị gửi event cảnh báo realtime lên hệ thống | Hoàn thành |

### Chưa triển khai

Các backlog bên dưới nằm ngoài phạm vi hiện tại:

| Backlog | Yêu cầu | Trạng thái |
| --- | --- | --- |
| 3 | Client xem/lọc danh sách cảnh báo theo thiết bị, mức độ nghiêm trọng và thời gian | Future work |
| 4 | Hệ thống tự nâng cảnh báo lên critical khi có nhiều event cùng loại trong 60 giây | Future work |
| 5 | Client tìm kiếm cảnh báo theo nội dung hoặc tên/ID thiết bị | Future work |

---

## Công Nghệ Sử Dụng

| Hạng mục | Công nghệ |
| --- | --- |
| Ngôn ngữ | Go |
| HTTP framework | Gin |
| Database | PostgreSQL |
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

---

## Chạy Project Bằng Docker

Khởi động PostgreSQL, Adminer, chạy migrations và start API:

```bash
make dev-up
```

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

Project này cố ý chỉ triển khai Backlog 1 và Backlog 2.

Chưa bao gồm trong phạm vi hiện tại:

- API xem/lọc danh sách alert theo device, severity và thời gian của Backlog 3.
- Tự động nâng cảnh báo lên critical khi event lặp lại của Backlog 4.
- Tìm kiếm alert theo nội dung hoặc tên/ID device của Backlog 5.
- Rate limiting, idempotency key, partitioning bảng alert cho production traffic rất lớn.
- Production Docker image hardening.
- CI/CD pipeline.

---

## Ghi Chú Nộp Bài

- API đã có Swagger và có thể test trực tiếp bằng Swagger UI.
- Development environment có seed demo client để reviewer test nhanh.
- Database schema dùng `clients`, `client_tokens`, `devices`, và `alerts` để giữ mental model rõ ràng cho Backlog 1 và Backlog 2.
- `client_tokens` lưu auth session/refresh-token metadata; raw refresh token không bao giờ được lưu trực tiếp.
- `alerts` lưu realtime event append-only; realtime delivery dùng PostgreSQL LISTEN/NOTIFY và SSE.
- Device ingest dùng device API key, còn client stream dùng JWT access token.
