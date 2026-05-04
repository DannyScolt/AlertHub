package middleware

import (
	"net/http"

	"alerthub/core/config"
	deviceRepo "alerthub/core/repository/device"
	"alerthub/core/utils/apikey"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
)

const DeviceIDKey = "device_id"

func DeviceAuth(cfg *config.Config, repo deviceRepo.DeviceRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		rawKey := authorizationValue(header)
		if rawKey == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid device API key", nil)
			c.Abort()
			return
		}

		device, err := repo.FindByAPIKeyHash(c.Request.Context(), apikey.Hash(rawKey))
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid device API key", nil)
			c.Abort()
			return
		}

		c.Set(DeviceIDKey, device.ID)
		c.Set(ClientIDKey, device.ClientID)
		c.Next()
	}
}
