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
// @description AlertHub IoT device management API for Backlog 1.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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
