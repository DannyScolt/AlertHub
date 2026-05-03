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
// @description AlertHub IoT device management and realtime alert ingestion API for Backlog 1 and Backlog 2.
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

	cfg := config.LoadConfig()
	db, err := database.NewPostgresPool(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	server.Run(cfg, db)
}
