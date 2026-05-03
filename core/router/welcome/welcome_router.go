package welcome

import (
	welcomeHandler "alerthub/core/handlers/welcome"

	"github.com/gin-gonic/gin"
)

func SetupWelcomeRoutes(router *gin.Engine) {
	handler := welcomeHandler.NewWelcomeHandler()

	group := router.Group("/api/v1")
	group.GET("/health", handler.Health)
}
