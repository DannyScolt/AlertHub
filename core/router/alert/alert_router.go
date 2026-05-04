package alert

import (
	"alerthub/core/config"
	alertHandler "alerthub/core/handlers/alert"
	"alerthub/core/middleware"
	alertRepo "alerthub/core/repository/alert"
	deviceRepo "alerthub/core/repository/device"
	alertService "alerthub/core/services/alert"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupAlertRoutes(router *gin.Engine, cfg *config.Config, db *pgxpool.Pool, streamService alertService.StreamService) {
	alertRepository := alertRepo.NewAlertRepository(db)
	alertNotifier := alertRepo.NewNotifier(db)
	deviceRepository := deviceRepo.NewDeviceRepository(db)
	ingestService := alertService.NewIngestService(alertRepository, alertNotifier)
	queryService := alertService.NewQueryService(alertRepository)
	ingestHandler := alertHandler.NewIngestHandler(ingestService)
	queryHandler := alertHandler.NewQueryHandler(queryService)
	streamHandler := alertHandler.NewStreamHandler(streamService)

	events := router.Group("/api/v1/events")
	events.Use(middleware.DeviceAuth(cfg, deviceRepository))
	events.POST("", ingestHandler.Ingest)
	events.POST("/batch", ingestHandler.IngestBatch)

	alerts := router.Group("/api/v1/alerts")
	alerts.Use(middleware.Auth(cfg))
	alerts.GET("", queryHandler.List)
	alerts.GET("/stream", streamHandler.Stream)
}
