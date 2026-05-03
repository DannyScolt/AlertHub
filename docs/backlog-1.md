# Backlog 1 — Client đăng ký và quản lý thiết bị

Backlog 1 tập trung vào luồng client đăng nhập, đăng ký thiết bị IoT mới và truy vấn danh sách thiết bị theo trạng thái.

> Yêu cầu: Là một client, tôi có thể đăng ký thiết bị mới vào hệ thống và truy vấn danh sách thiết bị theo trạng thái.

---

## Mục tiêu đã triển khai

| Hạng mục | Trạng thái |
| --- | --- |
| Client có thể đăng nhập để lấy JWT access token | Hoàn thành |
| Client có thể đăng ký thiết bị mới | Hoàn thành |
| API trả raw device API key đúng một lần khi tạo thiết bị | Hoàn thành |
| Client có thể xem danh sách thiết bị có pagination | Hoàn thành |
| Client có thể lọc danh sách thiết bị theo status | Hoàn thành |
| Client có thể xem chi tiết, cập nhật, soft-delete, restore thiết bị | Hoàn thành |
| Client có thể rotate device API key | Hoàn thành |
| Swagger có mô tả endpoint và request/response để reviewer test | Hoàn thành |

---

## Thành phần liên quan

```text
Client
  │
  │ 1. Login bằng email/password
  ▼
Auth API
  │
  │ 2. Trả JWT access_token
  ▼
Device API
  │
  │ 3. Client dùng Authorization: Bearer <access_token>
  ▼
PostgreSQL
  ├── clients
  └── devices
```

Các bảng chính:

```text
clients
└── Lưu thông tin client đăng ký hệ thống, email, name và password hash.

devices
└── Lưu thiết bị IoT thuộc về client, API key hash, type/status enum, tags, metadata, timestamps và soft-delete state.

alerts
└── Được dùng để tính last_seen_at từ alert mới nhất của device. Backlog 1 không lưu last_seen_at trực tiếp trong devices.
```

---

## Acceptance criteria

Reviewer có thể xem Backlog 1 là hoàn thành khi các điều kiện bên dưới đều đúng:

| Tiêu chí | Cách kiểm tra |
| --- | --- |
| Client đăng nhập được bằng demo credentials | `POST /auth/login` trả `access_token` và `refresh_token` |
| Protected device APIs bắt buộc client JWT | Gọi không có `Authorization` trả `401 Unauthorized` |
| Client tạo được device mới | `POST /devices` trả `201 Created` và có device `id` |
| API trả raw device API key đúng một lần | Response create/rotate có `api_key`; list/detail không trả raw key |
| Device API key không lưu raw trong DB | DB chỉ dùng `api_key_hash` |
| Client xem được danh sách device của mình | `GET /devices` trả data + pagination |
| Client lọc được device theo status | `GET /devices?status=active` chỉ trả device active |
| Client không truy cập được device của client khác | Cross-client request không lộ dữ liệu |
| Soft delete ẩn device khỏi list/detail mặc định | Sau `DELETE /devices/{id}`, list/detail không trả device như active |
| Restore đưa device về lại trạng thái dùng được | `POST /devices/{id}/restore` thành công và status về `inactive` |
| Rotate key trả key mới | `POST /devices/{id}/rotate-key` trả raw key mới |
| Swagger đủ để reviewer tự test | Swagger UI có request/response và auth scheme rõ ràng |

---

## Authentication cho Backlog 1

Các API device yêu cầu client JWT access token.

Header bắt buộc:

```http
Authorization: Bearer <access_token>
```

Token lấy từ API login:

```http
POST /api/v1/auth/login
```

Demo client ở môi trường development:

```text
Email: client@example.com
Password: password123
```

---

## Device Status

Các status hợp lệ:

```text
active
inactive
maintenance
error
```

Status được validate bằng enum cố định. Nếu gửi status không hợp lệ, API trả validation/business error tương ứng.

---

## Device Type

Các type hợp lệ:

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

---

## Dữ liệu request/response quan trọng

### Device create request

| Field | Kiểu | Bắt buộc | Ghi chú |
| --- | --- | --- | --- |
| `name` | string | Có | Tên device, unique theo từng client trong nhóm chưa soft-delete |
| `type` | string enum | Có | Một trong các Device Type hợp lệ |
| `status` | string enum | Có | Một trong các Device Status hợp lệ |
| `tags` | string array | Không | Nhãn để client phân loại device |
| `metadata` | object | Không | JSON metadata tự do, ví dụ location, firmware, serial |

### Device response

| Field | Ý nghĩa |
| --- | --- |
| `id` | UUID của device |
| `name` | Tên device |
| `type` | Loại device |
| `status` | Trạng thái hiện tại |
| `tags` | Danh sách tag |
| `metadata` | Metadata dạng JSON |
| `api_key` | Chỉ có trong create/rotate response, không xuất hiện ở list/detail |
| `last_seen_at` | Thời điểm alert mới nhất của device, có thể chưa có nếu device chưa gửi event |
| `created_at` | Thời điểm tạo device |
| `updated_at` | Thời điểm cập nhật gần nhất |

### Pagination response

`GET /devices` trả thêm object `pagination`:

| Field | Ý nghĩa |
| --- | --- |
| `page` | Trang hiện tại |
| `page_size` | Số item mỗi trang |
| `total` | Tổng số device match filter |
| `total_pages` | Tổng số trang |
| `has_next` | Có trang tiếp theo hay không |
| `has_previous` | Có trang trước hay không |

---

## API đã triển khai

Base URL:

```text
http://localhost:8080/api/v1
```

| Method | Endpoint | Mục đích | Auth |
| --- | --- | --- | --- |
| POST | `/auth/login` | Đăng nhập client và lấy token | Không |
| GET | `/clients/me` | Xem profile client hiện tại | Client JWT |
| POST | `/devices` | Đăng ký thiết bị mới | Client JWT |
| GET | `/devices` | Xem danh sách thiết bị, có filter và pagination | Client JWT |
| GET | `/devices/{id}` | Xem chi tiết một thiết bị | Client JWT |
| PATCH | `/devices/{id}` | Cập nhật thiết bị | Client JWT |
| DELETE | `/devices/{id}` | Soft-delete thiết bị | Client JWT |
| POST | `/devices/{id}/restore` | Restore thiết bị đã soft-delete | Client JWT |
| POST | `/devices/{id}/rotate-key` | Rotate device API key | Client JWT |

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

Response mong đợi: `200 OK`

```json
{
  "status": true,
  "message": "Login successful",
  "data": {
    "access_token": "...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

Copy `data.access_token`, sau đó trong Swagger bấm **Authorize** và nhập:

```text
Bearer <access_token>
```

---

### 2. Xem thông tin client hiện tại

```http
GET /api/v1/clients/me
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

```json
{
  "status": true,
  "message": "Client retrieved successfully",
  "data": {
    "id": "...",
    "email": "client@example.com",
    "name": "Demo Client",
    "created_at": "2026-05-03T12:00:00Z",
    "updated_at": "2026-05-03T12:00:00Z"
  }
}
```

---

### 3. Đăng ký thiết bị mới

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

Response mong đợi: `201 Created`

```json
{
  "status": true,
  "message": "Device created successfully",
  "data": {
    "id": "4d285f4b-2a87-4a86-a5b8-05b09c6d1234",
    "name": "Warehouse Temperature Sensor",
    "type": "temperature_sensor",
    "status": "active",
    "tags": ["warehouse", "floor-1"],
    "metadata": {
      "location": "Room 101"
    },
    "api_key": "ah_dev_xxx",
    "created_at": "2026-05-03T12:00:00Z",
    "updated_at": "2026-05-03T12:00:00Z"
  }
}
```

Lưu lại `data.id` để test các endpoint tiếp theo.

Nếu cần dùng Backlog 2, lưu thêm `data.api_key`. Raw device API key chỉ được trả về đúng một lần khi tạo device hoặc rotate key.

---

### 4. Xem danh sách thiết bị

```http
GET /api/v1/devices?page=1&page_size=20
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

```json
{
  "status": true,
  "message": "Devices retrieved successfully",
  "data": [
    {
      "id": "4d285f4b-2a87-4a86-a5b8-05b09c6d1234",
      "name": "Warehouse Temperature Sensor",
      "type": "temperature_sensor",
      "status": "active",
      "tags": ["warehouse", "floor-1"],
      "metadata": {
        "location": "Room 101"
      },
      "created_at": "2026-05-03T12:00:00Z",
      "updated_at": "2026-05-03T12:00:00Z"
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

`last_seen_at` có thể chưa xuất hiện nếu device chưa từng gửi alert. Sau khi device gửi alert ở Backlog 2, field này sẽ được tính từ alert mới nhất.

---

### 5. Lọc danh sách thiết bị theo trạng thái

```http
GET /api/v1/devices?status=active&page=1&page_size=20
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`, chỉ gồm các device có `status = active`.

Endpoint này là phần chính để chứng minh yêu cầu Backlog 1: client truy vấn danh sách thiết bị theo trạng thái.

Có thể test thêm các status:

```http
GET /api/v1/devices?status=inactive&page=1&page_size=20
GET /api/v1/devices?status=maintenance&page=1&page_size=20
GET /api/v1/devices?status=error&page=1&page_size=20
```

---

### 6. Xem chi tiết thiết bị

```http
GET /api/v1/devices/{id}
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

```json
{
  "status": true,
  "message": "Device retrieved successfully",
  "data": {
    "id": "4d285f4b-2a87-4a86-a5b8-05b09c6d1234",
    "name": "Warehouse Temperature Sensor",
    "type": "temperature_sensor",
    "status": "active",
    "tags": ["warehouse", "floor-1"],
    "metadata": {
      "location": "Room 101"
    },
    "created_at": "2026-05-03T12:00:00Z",
    "updated_at": "2026-05-03T12:00:00Z"
  }
}
```

---

### 7. Cập nhật thiết bị

```http
PATCH /api/v1/devices/{id}
Authorization: Bearer <access_token>
Content-Type: application/json
```

Request ví dụ:

```json
{
  "name": "Warehouse Temperature Sensor Updated",
  "status": "maintenance",
  "tags": ["warehouse", "floor-1", "calibration"],
  "metadata": {
    "location": "Room 101",
    "note": "Calibrating"
  }
}
```

Response mong đợi: `200 OK`, device được cập nhật.

---

### 8. Rotate device API key

```http
POST /api/v1/devices/{id}/rotate-key
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

```json
{
  "status": true,
  "message": "Device API key rotated successfully",
  "data": {
    "api_key": "ah_dev_new_xxx"
  }
}
```

Raw API key mới chỉ hiển thị trong response này. API chỉ lưu hash, không lưu raw key.

---

### 9. Soft-delete thiết bị

```http
DELETE /api/v1/devices/{id}
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

Sau khi soft-delete:

- Device bị ẩn khỏi list/detail mặc định.
- Device không thể update.
- Device không thể rotate API key.
- Device API key không còn dùng được để gửi event.

---

### 10. Restore thiết bị

```http
POST /api/v1/devices/{id}/restore
Authorization: Bearer <access_token>
```

Response mong đợi: `200 OK`

Sau restore, device quay lại trạng thái `inactive`.

---

## Business rules

- Một device thuộc về đúng một client.
- Client chỉ thấy và thao tác được device của chính mình.
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

## Negative cases nên kiểm tra

| Case | Kết quả mong đợi |
| --- | --- |
| Gọi device API không có access token | `401 Unauthorized` |
| Gọi device API bằng token sai/expired | `401 Unauthorized` |
| Tạo device thiếu name/type/status | `400 Bad Request` |
| Tạo device với type không hợp lệ | `400 Bad Request` hoặc business validation error |
| Tạo device với status không hợp lệ | `400 Bad Request` hoặc business validation error |
| Tạo trùng tên device trong cùng client | Conflict/business error |
| Lọc device bằng status không hợp lệ | `400 Bad Request` hoặc business validation error |
| Xem device không thuộc client hiện tại | `404 Not Found` hoặc unauthorized scoped response |
| Update device đã soft-delete | Business error |
| Rotate key của device đã soft-delete | Business error |

---

## Checklist reviewer cho Backlog 1

Có thể tick theo thứ tự này khi review:

- [ ] Chạy `make dev-up` thành công.
- [ ] Mở Swagger tại `http://localhost:8080/swagger/index.html`.
- [ ] Login bằng `client@example.com / password123` thành công.
- [ ] Authorize Swagger bằng `Bearer <access_token>`.
- [ ] Gọi `GET /clients/me` thấy đúng client hiện tại.
- [ ] Tạo device mới bằng `POST /devices` và lưu lại `id`, `api_key`.
- [ ] Gọi `GET /devices` thấy device vừa tạo.
- [ ] Gọi `GET /devices?status=active` thấy device active.
- [ ] Gọi filter status khác để xác nhận API lọc đúng.
- [ ] Gọi `GET /devices/{id}` thấy chi tiết device.
- [ ] Gọi `PATCH /devices/{id}` đổi status/name/tags/metadata thành công.
- [ ] Gọi `POST /devices/{id}/rotate-key` nhận raw API key mới.
- [ ] Gọi `DELETE /devices/{id}` soft-delete thành công.
- [ ] Xác nhận device đã soft-delete không còn xuất hiện như device active bình thường.
- [ ] Gọi `POST /devices/{id}/restore` thành công và device về `inactive`.
- [ ] Gọi API không có access token để xác nhận `401 Unauthorized`.

---

## Mapping yêu cầu đề bài sang implementation

| Yêu cầu đề bài | Implementation trong project |
| --- | --- |
| Client đăng ký thiết bị mới | `POST /api/v1/devices` |
| Thiết bị thuộc về client | Device record có `client_id`, lấy từ JWT access token |
| Client xem danh sách thiết bị | `GET /api/v1/devices` |
| Client truy vấn theo trạng thái | Query `status` trong `GET /api/v1/devices?status=active` |
| Bảo vệ dữ liệu giữa các client | Repository/service scope theo `client_id` |
| Device có API key để dùng ở Backlog 2 | `api_key` trả một lần khi create/rotate, DB lưu hash |

---

## Ghi chú cho reviewer

- Backlog 1 không yêu cầu API xem alert; phần đó thuộc future Backlog 3.
- `last_seen_at` là field dẫn xuất từ bảng `alerts`, không phải cột trong `devices`.
- Nếu device chưa gửi alert, `last_seen_at` có thể chưa xuất hiện trong JSON response do giá trị nil.
- Raw `api_key` không xuất hiện ở list/detail để tránh lộ secret.
- Swagger là cách test nhanh nhất vì đã có sẵn authorize button cho Bearer token.

---

## Cách chạy local để test Backlog 1

Khởi động môi trường:

```bash
make dev-up
```

Mở Swagger:

```text
http://localhost:8080/swagger/index.html
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
