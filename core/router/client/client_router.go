package client

import (
	"alerthub/core/config"
	clientHandler "alerthub/core/handlers/client"
	"alerthub/core/middleware"
	clientRepo "alerthub/core/repository/client"
	clientService "alerthub/core/services/client"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupClientRoutes(router *gin.Engine, cfg *config.Config, db *pgxpool.Pool) {
	clientRepository := clientRepo.NewClientRepository(db)
	service := clientService.NewClientService(clientRepository)
	handler := clientHandler.NewClientHandler(service)

	group := router.Group("/api/v1/clients")
	group.Use(middleware.Auth(cfg))
	group.GET("/me", handler.Me)
}
