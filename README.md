# AlertHub API

AlertHub is a Go REST API implementation for the Backend Coding Challenge. This submission focuses on **Backlog 1**, the highest-priority requirement from the challenge:

> As a client, I can register a new device in the system and retrieve the device list by status.

The project includes client authentication, PostgreSQL persistence, Swagger documentation, Docker-based local development, and a clean layered structure so reviewers can run and test the API quickly.

---

## Backlog Scope

### Implemented

| Backlog | Requirement | Status |
| --- | --- | --- |
| 1 | Client can register a new device and list devices by status | Complete |

### Not Implemented

The following backlog items are intentionally out of scope for this Backlog 1 submission:

| Backlog | Requirement | Status |
| --- | --- | --- |
| 2 | Device sends realtime alert events | Future work |
| 3 | Client views and filters alerts by device, severity, and time range | Future work |
| 4 | System auto-escalates repeated alerts to critical | Future work |
| 5 | Client searches alerts by content keyword or device name/ID | Future work |

---

## Tech Stack

| Area | Technology |
| --- | --- |
| Language | Go |
| HTTP framework | Gin |
| Database | PostgreSQL |
| DB driver | pgx / pgxpool |
| Auth | JWT access tokens + opaque refresh tokens |
| Password hashing | bcrypt |
| API docs | Swagger / OpenAPI via swaggo |
| Local runtime | Docker Compose |
| DB inspection | Adminer |
| Hot reload | Air |

---

## Architecture

```text
cmd/api                  # API entrypoint
core/config              # Environment configuration
core/database            # PostgreSQL connection
core/domain              # Domain models and enums
core/dto                 # HTTP request/response DTOs
core/repository          # PostgreSQL repositories
core/services            # Business logic
core/handlers            # Gin HTTP handlers
core/router              # Route registration and dependency wiring
core/middleware          # Global and auth middleware
core/server              # Gin server bootstrap
core/utils               # Shared helpers
migrations               # PostgreSQL migrations
docs                     # Generated Swagger docs
```

Layering:

```text
router -> handler -> service -> repository -> PostgreSQL
```

`core/server` only bootstraps Gin and calls the router aggregator. Repository/service/handler wiring lives inside module routers, matching the reference-style project structure.

---

## Database Structure

```text
clients
└── Registered API clients. Includes nullable remember_token for familiarity, but refresh tokens are not stored here.

client_tokens
└── Client auth sessions, refresh-token hashes, rotation chains, revoke/logout state, user agent, and IP metadata.

devices
└── IoT devices owned by clients. Stores API key hashes, type/status enums, tags, metadata, timestamps, and soft-delete state.
```

Important security decisions:

- Passwords are stored as bcrypt hashes.
- Refresh tokens are stored only as hashes in `client_tokens`.
- Device API keys are stored only as hashes.
- Raw device API keys are returned only once on create or rotate.
- Auth responses return only token values, not client profile data.

---

## Prerequisites

- Go 1.25+
- Docker and Docker Compose
- Make

Optional for local development without Docker:

- PostgreSQL 16+

---

## Environment Setup

Copy the sample environment file:

```bash
cp .env.example .env
```

For local host execution, use `localhost` in `DATABASE_URL`:

```env
DATABASE_URL=postgres://alerthub:alerthub@localhost:5432/alerthub?sslmode=disable
```

Docker Compose overrides the API container database URL to use the Docker service name `postgres`.

---

## Run Locally with Docker

Start PostgreSQL, Adminer, run migrations, and start the API:

```bash
make dev-up
```

This runs:

```text
1. docker compose up -d postgres adminer
2. docker compose run --rm migrate
3. docker compose up -d api
```

The API will be available at:

```text
http://localhost:8080
```

Stop containers:

```bash
make docker-down
```

View logs:

```bash
make docker-logs
```

Reset local Docker database data:

```bash
docker compose down -v
make dev-up
```

---

## Swagger

Open Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

Regenerate Swagger docs after changing handlers or DTOs:

```bash
make swagger
```

---

## Adminer

Open Adminer:

```text
http://localhost:8081
```

Use these credentials:

```text
System: PostgreSQL
Server: postgres
Username: alerthub
Password: alerthub
Database: alerthub
```

Inside Adminer, use `postgres` as the server. Do not use `localhost`, because Adminer runs inside its own Docker container.

---

## Demo Client

In development, the API seeds a demo client for quick Swagger testing:

```text
Email: client@example.com
Password: password123
```

The seed is skipped when `APP_ENV=production`.

---

## Backlog 1 Reviewer Flow

Use this flow to verify the implemented challenge requirement.

### 1. Login

```http
POST /api/v1/auth/login
```

Request:

```json
{
  "email": "client@example.com",
  "password": "password123"
}
```

Response contains:

```json
{
  "data": {
    "access_token": "...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

Copy `access_token`, then click **Authorize** in Swagger and enter:

```text
Bearer <access_token>
```

### 2. Register a New Device

```http
POST /api/v1/devices
Authorization: Bearer <access_token>
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

Expected response: `201 Created`

```json
{
  "status": true,
  "message": "Device created successfully",
  "data": {
    "id": "4d285f4b-2a87-4a86-a5b8-05b09c6d1234",
    "name": "Warehouse Temperature Sensor",
    "type": "temperature_sensor",
    "status": "active",
    "api_key": "ah_dev_xxx",
    "created_at": "2026-05-03T12:00:00Z",
    "updated_at": "2026-05-03T12:00:00Z"
  }
}
```

Save `data.api_key` if needed. It is returned only once.

### 3. List Devices

```http
GET /api/v1/devices?page=1&page_size=20
Authorization: Bearer <access_token>
```

Expected response: `200 OK`

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
      "last_seen_at": null,
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

### 4. List Devices by Status

```http
GET /api/v1/devices?status=active&page=1&page_size=20
Authorization: Bearer <access_token>
```

Allowed statuses:

```text
active
inactive
maintenance
error
```

This endpoint satisfies the Backlog 1 query requirement.

---

## API Reference

Base URL:

```text
http://localhost:8080/api/v1
```

### Health

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/health` | Check API health |

### Auth

| Method | Endpoint | Description |
| --- | --- | --- |
| POST | `/auth/register` | Register a new client and issue tokens |
| POST | `/auth/login` | Login and issue tokens |
| POST | `/auth/refresh` | Rotate refresh token and issue new tokens |
| POST | `/auth/logout` | Logout one session by refresh token |
| POST | `/auth/logout-all` | Logout all sessions for current client |
| GET | `/auth/sessions` | List current client's sessions |
| DELETE | `/auth/sessions/{id}` | Revoke one session by session ID |

### Client

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/clients/me` | Get authenticated client profile |

### Devices

| Method | Endpoint | Description |
| --- | --- | --- |
| POST | `/devices` | Register a new device |
| GET | `/devices` | List devices with optional filters and pagination |
| GET | `/devices/{id}` | Get one device |
| PATCH | `/devices/{id}` | Update one device |
| DELETE | `/devices/{id}` | Soft delete one device |
| POST | `/devices/{id}/restore` | Restore a soft-deleted device |
| POST | `/devices/{id}/rotate-key` | Rotate a device API key |

---

## Device Rules

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

- A device belongs to exactly one client.
- Device names must be unique per client among non-deleted devices.
- Device type and status are validated against fixed enums.
- Device delete is a soft delete using `deleted_at`.
- Soft-deleted devices are hidden by default.
- Deleted devices can be restored within the configured restore window.
- Restored devices are set to `inactive`.
- Deleted devices cannot be updated or have API keys rotated.
- Device API keys are returned only on create and rotate.
- Only device API key hashes are stored.

---

## Useful Commands

| Command | Description |
| --- | --- |
| `make dev-up` | Start Postgres, Adminer, migrations, and API |
| `make docker-up` | Start Docker services |
| `make docker-down` | Stop Docker services |
| `make docker-logs` | View API logs |
| `make migrate-up` | Run migrations through Docker |
| `make migrate-down` | Roll back migrations through Docker |
| `make swagger` | Regenerate Swagger docs |
| `make test` | Run `go test ./...` |
| `make build` | Run `go build ./...` |
| `make tidy` | Run `go mod tidy` |

---

## Verification

Before submitting or after changing code, run:

```bash
go test ./...
go build ./...
docker compose config --quiet
```

To verify a fresh database schema:

```bash
docker compose down -v
make dev-up
docker compose exec -T postgres psql -U alerthub -d alerthub -c "SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name;"
```

Expected tables:

```text
client_tokens
clients
devices
schema_migrations
```

---

## Known Limitations

This project intentionally implements Backlog 1 only.

Not included in this submission:

- Realtime device event ingestion.
- Alert storage/listing/filtering/search.
- Automatic critical escalation for repeated alert events.
- Production Docker image hardening.
- CI/CD pipeline.

---

## Submission Notes

- The API is documented in Swagger and can be tested directly from Swagger UI.
- The demo client is seeded in development for quick reviewer access.
- The database schema uses `clients`, `client_tokens`, and `devices` for a clean Backlog 1 mental model.
- `client_tokens` stores auth session/refresh-token metadata; raw refresh tokens are never stored.
