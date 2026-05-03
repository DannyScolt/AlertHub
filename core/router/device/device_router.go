package device

import (
	"alerthub/core/config"
	deviceHandler "alerthub/core/handlers/device"
	"alerthub/core/middleware"
	alertRepo "alerthub/core/repository/alert"
	deviceRepo "alerthub/core/repository/device"
	deviceService "alerthub/core/services/device"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupDeviceRoutes(router *gin.Engine, cfg *config.Config, db *pgxpool.Pool) {
	deviceRepository := deviceRepo.NewDeviceRepository(db)
	alertRepository := alertRepo.NewAlertRepository(db)
	service := deviceService.NewDeviceService(cfg, deviceRepository, alertRepository)
	handler := deviceHandler.NewDeviceHandler(service)

	group := router.Group("/api/v1/devices")
	group.Use(middleware.Auth(cfg))
	group.POST("", handler.Create)
	group.GET("", handler.List)
	group.GET("/:id", handler.Get)
	group.PATCH("/:id", handler.Update)
	group.DELETE("/:id", handler.Delete)
	group.POST("/:id/restore", handler.Restore)
	group.POST("/:id/rotate-key", handler.RotateAPIKey)
}
