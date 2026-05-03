package swagger

import (
	"alerthub/core/config"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupSwaggerRoutes(router *gin.Engine, cfg *config.Config) {
	if !cfg.SwaggerEnabled {
		return
	}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
