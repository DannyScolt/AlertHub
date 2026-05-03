package server

import (
	"context"
	"log"

	"alerthub/core/config"
	"alerthub/core/middleware"
	alertRepo "alerthub/core/repository/alert"
	"alerthub/core/router"
	alertService "alerthub/core/services/alert"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(cfg *config.Config, db *pgxpool.Pool) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	streamService := alertService.NewStreamService()
	alertRepository := alertRepo.NewAlertRepository(db)
	listener := alertService.NewAlertListener(db, alertRepository, streamService)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go listener.Run(ctx)

	ginRouter := gin.New()
	middleware.RegisterGlobal(ginRouter)
	router.SetupRouter(ginRouter, cfg, db, streamService)

	log.Printf("Starting AlertHub API on port %s", cfg.HTTPPort)
	if err := ginRouter.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
