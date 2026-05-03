package server

import (
	"log"

	"alerthub/core/config"
	"alerthub/core/middleware"
	"alerthub/core/router"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(cfg *config.Config, db *pgxpool.Pool) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	ginRouter := gin.New()
	middleware.RegisterGlobal(ginRouter)
	router.SetupRouter(ginRouter, cfg, db)

	log.Printf("Starting AlertHub API on port %s", cfg.HTTPPort)
	if err := ginRouter.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
