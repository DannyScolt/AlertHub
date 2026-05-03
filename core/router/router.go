package router

import (
	"alerthub/core/config"
	authRouter "alerthub/core/router/auth"
	clientRouter "alerthub/core/router/client"
	deviceRouter "alerthub/core/router/device"
	swaggerRouter "alerthub/core/router/swagger"
	welcomeRouter "alerthub/core/router/welcome"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(router *gin.Engine, cfg *config.Config, db *pgxpool.Pool) {
	welcomeRouter.SetupWelcomeRoutes(router)
	authRouter.SetupAuthRoutes(router, cfg, db)
	clientRouter.SetupClientRoutes(router, cfg, db)
	deviceRouter.SetupDeviceRoutes(router, cfg, db)
	swaggerRouter.SetupSwaggerRoutes(router, cfg)
}
