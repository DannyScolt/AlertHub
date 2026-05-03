# AlertHub

AlertHub is a Go REST API for the Backend Coding Challenge Backlog 1 scope. It focuses on client authentication, refresh-token session management, client profile access, and IoT device registration/query/management backed by PostgreSQL.

## Tech Stack

- Go + Gin
- PostgreSQL + pgx
- JWT access tokens
- Opaque refresh tokens stored as hashes
- bcrypt password hashing
- Swagger/OpenAPI via swaggo
- Docker Compose for local development
- Adminer for database inspection

## Project Structure

```text
cmd/api                  # API entrypoint
core/config              # Environment configuration
core/database            # PostgreSQL connection
core/domain              # Domain models/enums
core/dto                 # HTTP request/response DTOs
core/repository          # PostgreSQL repositories
core/services            # Business logic
core/handlers            # Gin HTTP handlers
core/router              # Route registration and dependency wiring
core/middleware          # Global/auth middleware
core/server              # Gin server bootstrap
core/utils               # Shared helpers
migrations               # PostgreSQL migrations
docs                     # Generated Swagger docs
```

The server only bootstraps Gin and calls the router aggregator. Repository/service/handler wiring lives inside each module router to match the reference structure style.

## Environment

Copy the example environment file:

```bash
cp .env.example .env
```

For local host execution, `DATABASE_URL` should point to `localhost`:

```env
DATABASE_URL=postgres://alerthub:alerthub@localhost:5432/alerthub?sslmode=disable
```

Docker Compose overrides the API container database URL to use the Docker service name `postgres`.

## Run with Docker

Start PostgreSQL, Adminer, and the API:

```bash
make docker-up
```

Run database migrations:

```bash
make migrate-up
```

Or run the full local dev startup flow:

```bash
make dev-up
```

Stop containers:

```bash
make docker-down
```

View API logs:

```bash
make docker-logs
```

## Adminer

Open:

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

Important: inside Adminer, use `postgres` as the server. Do not use `localhost`, because Adminer runs inside its own Docker container.

## Swagger

Open:

```text
http://localhost:8080/swagger/index.html
```

Regenerate Swagger docs after changing annotations or DTOs:

```bash
make swagger
```

## Demo Client

In development, the app seeds a demo client for quick Swagger/API testing:

```text
Email: client@example.com
Password: password123
```

The seed is skipped when `APP_ENV=production`.

## API List

Base URL:

```text
http://localhost:8080/api/v1
```

### Health

```text
GET /health
```

Checks whether the API is running.

### Auth

```text
POST /auth/register
POST /auth/login
POST /auth/refresh
POST /auth/logout
POST /auth/logout-all
GET /auth/sessions
DELETE /auth/sessions/{id}
```

Auth supports JWT access tokens and refresh-token sessions. Refresh tokens are opaque raw tokens returned to the client once and stored in the database only as hashes.

### Client

```text
GET /clients/me
```

Returns the currently authenticated client profile.

### Devices

```text
POST /devices
GET /devices
GET /devices/{id}
PATCH /devices/{id}
DELETE /devices/{id}
POST /devices/{id}/restore
POST /devices/{id}/rotate-key
```

Device list supports filtering and pagination:

```text
GET /devices?status=active&type=temperature_sensor&page=1&page_size=20
```

Soft-deleted devices are hidden by default. To include them:

```text
GET /devices?include_deleted=true
```

## Device Rules

- Device belongs to one client.
- Device names must be unique per client among non-deleted devices.
- Device type and status are PostgreSQL enums.
- Device delete is soft delete via `deleted_at`.
- Deleted devices can be restored within the configured restore window.
- Restored devices are set to `inactive`.
- Device API keys are returned only on create and rotate.
- Only API key hashes are stored.

## Useful Commands

```bash
make test        # go test ./...
make build       # go build ./...
make tidy        # go mod tidy
make swagger     # regenerate Swagger docs
make migrate-up  # run migrations with Docker
make migrate-down
```

## Verification

Before submitting or after changing code, run:

```bash
go test ./...
go build ./...
```
