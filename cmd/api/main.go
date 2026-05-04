package main

import (
	"log"

	"alerthub/core/config"
	"alerthub/core/database"
	"alerthub/core/server"
	_ "alerthub/docs"
)

// @title AlertHub API
// @version 1.0
// @description AlertHub IoT device management and realtime alert API. Covers device registration (Backlog 1), realtime ingest with SSE (Backlog 2), alert query/filter/pagination (Backlog 3), auto-escalation with Redis cooldown (Backlog 4), and alert search (Backlog 5). Use Authorize -> BearerAuth for client JWT (from /auth/login). Use Authorize -> DeviceAPIKey for device API key (from POST /devices). Swagger UI can paste the raw token/key directly; curl can use either Authorization: <token> or Authorization: Bearer <token>.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Client JWT access token from /auth/login. In Swagger UI, paste data.access_token directly. In curl, use either Authorization: <access_token> or Authorization: Bearer <access_token>.
// @securityDefinitions.apikey DeviceAPIKey
// @in header
// @name Authorization
// @description Device API key returned once when creating or rotating a device. In Swagger UI, paste ah_dev_... directly. In curl, use either Authorization: ah_dev_... or Authorization: Bearer ah_dev_...
func main() {
	log.Println("Starting AlertHub API...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	db, err := database.NewPostgresPool(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	server.Run(cfg, db)
}
