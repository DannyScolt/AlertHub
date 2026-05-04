package server

import (
	"context"
	"log"
	"time"

	"alerthub/core/config"
	redisInfra "alerthub/core/infra/redis"
	"alerthub/core/middleware"
	alertRepo "alerthub/core/repository/alert"
	escalationRepo "alerthub/core/repository/escalation"
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

	redisClient, err := redisInfra.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	cooldownStore := escalationRepo.NewRedisCooldownStore(redisClient)
	escalationService := alertService.NewEscalationService(
		alertRepository,
		alertRepository,
		alertRepository,
		alertRepo.NewNotifier(db),
		cooldownStore,
		alertService.EscalationConfig{Enabled: cfg.EscalationEnabled, Threshold: cfg.EscalationThreshold, Window: cfg.EscalationWindow, Cooldown: cfg.EscalationCooldown},
		func() time.Time { return time.Now().UTC() },
	)
	escalationListener := alertService.NewEscalationListener(db, escalationService)
	go escalationListener.Run(ctx)

	ginRouter := gin.New()
	middleware.RegisterGlobal(ginRouter)
	router.SetupRouter(ginRouter, cfg, db, streamService)

	log.Printf("Starting AlertHub API on port %s", cfg.HTTPPort)
	if err := ginRouter.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
