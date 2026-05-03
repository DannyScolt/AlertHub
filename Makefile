.PHONY: dev-up docker-up docker-down docker-logs migrate-up migrate-down swagger test build tidy

dev-up:
	docker compose up -d postgres adminer
	docker compose run --rm migrate
	docker compose up -d api

docker-up:
	docker compose up -d postgres adminer api

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

migrate-up:
	docker compose run --rm migrate

migrate-down:
	./scripts/migrate-down.sh

swagger:
	swag init -g cmd/api/main.go -o docs --parseInternal

test:
	go test ./...

build:
	go build ./...

tidy:
	go mod tidy
