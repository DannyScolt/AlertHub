package welcome

import (
	"net/http"

	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
)

type WelcomeHandler interface{ Health(*gin.Context) }

type welcomeHandler struct{}

func NewWelcomeHandler() WelcomeHandler { return &welcomeHandler{} }

// Health godoc
// @Summary Health check
// @Tags Health
// @Produce json
// @Success 200 {object} common.APIResponse
// @Router /health [get]
func (h *welcomeHandler) Health(c *gin.Context) {
	response.Success(c, http.StatusOK, "AlertHub API is healthy", gin.H{"service": "alerthub", "status": "ok"})
}
