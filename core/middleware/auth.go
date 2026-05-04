package middleware

import (
	"net/http"
	"strings"

	"alerthub/core/config"
	"alerthub/core/utils/response"
	"alerthub/core/utils/token"

	"github.com/gin-gonic/gin"
)

const ClientIDKey = "client_id"

func authorizationValue(header string) string {
	header = strings.TrimSpace(header)
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		rawToken := authorizationValue(header)
		if rawToken == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid authorization header", nil)
			c.Abort()
			return
		}
		claims, err := token.VerifyAccessToken(rawToken, cfg.JWTAccessSecret)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid access token", nil)
			c.Abort()
			return
		}
		c.Set(ClientIDKey, claims.ClientID)
		c.Next()
	}
}
