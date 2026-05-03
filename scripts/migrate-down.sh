#!/bin/sh
set -e

DATABASE_URL="postgres://alerthub:alerthub@postgres:5432/alerthub?sslmode=disable"
docker compose run --rm --entrypoint migrate migrate -path /migrations -database "$DATABASE_URL" down 1
