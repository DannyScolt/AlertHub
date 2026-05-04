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
// @description AlertHub IoT device management and realtime alert API. Covers device registration (Backlog 1), realtime ingest with SSE (Backlog 2), alert query/filter/pagination (Backlog 3), auto-escalation with Redis cooldown (Backlog 4), and alert search (Backlog 5). Use Authorize -> BearerAuth for client JWT (from /auth/login). Use Authorize -> DeviceAPIKey for device API key (from POST /devices).
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Client JWT access token. Use value format: Bearer <access_token>.
// @securityDefinitions.apikey DeviceAPIKey
// @in header
// @name Authorization
// @description Device API key returned once when creating or rotating a device. Use value format: Bearer ah_dev_...
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
