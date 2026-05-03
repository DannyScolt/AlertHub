package auth

import (
	"context"
	"log"

	"alerthub/core/config"
	clientDomain "alerthub/core/domain/client"
	authHandler "alerthub/core/handlers/auth"
	"alerthub/core/middleware"
	clientRepo "alerthub/core/repository/client"
	clientTokenRepo "alerthub/core/repository/client_token"
	authService "alerthub/core/services/auth"
	"alerthub/core/utils/password"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sampleClientName     = "Demo Client"
	sampleClientEmail    = "client@example.com"
	sampleClientPassword = "password123"
)

func SetupAuthRoutes(router *gin.Engine, cfg *config.Config, db *pgxpool.Pool) {
	clientRepository := clientRepo.NewClientRepository(db)
	seedSampleClient(context.Background(), cfg, clientRepository)
	clientTokenRepository := clientTokenRepo.NewClientTokenRepository(db)
	service := authService.NewAuthService(cfg, clientRepository, clientTokenRepository)
	handler := authHandler.NewAuthHandler(service)

	group := router.Group("/api/v1/auth")
	group.POST("/register", handler.Register)
	group.POST("/login", handler.Login)
	group.POST("/refresh", handler.Refresh)

	protected := group.Group("")
	protected.Use(middleware.Auth(cfg))
	protected.POST("/logout", handler.Logout)
	protected.POST("/logout-all", handler.LogoutAll)
	protected.GET("/sessions", handler.Sessions)
	protected.DELETE("/sessions/:id", handler.RevokeSession)
}

func seedSampleClient(ctx context.Context, cfg *config.Config, repo clientRepo.ClientRepository) {
	if cfg.AppEnv == "production" {
		return
	}

	exists, err := repo.EmailExists(ctx, sampleClientEmail)
	if err != nil || exists {
		if err != nil {
			log.Printf("Skipping sample client seed: %v", err)
		}
		return
	}

	hash, err := password.Hash(sampleClientPassword)
	if err != nil {
		log.Printf("Skipping sample client seed: %v", err)
		return
	}

	_, err = repo.Create(ctx, clientDomain.Client{Name: sampleClientName, Email: sampleClientEmail, PasswordHash: hash})
	if err != nil {
		log.Printf("Skipping sample client seed: %v", err)
		return
	}

	log.Printf("Seeded sample client for Swagger login: %s / %s", sampleClientEmail, sampleClientPassword)
}
