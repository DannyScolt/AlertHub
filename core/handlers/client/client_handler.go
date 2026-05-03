package client

import (
	"net/http"

	"alerthub/core/middleware"
	clientService "alerthub/core/services/client"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClientHandler interface{ Me(*gin.Context) }

type clientHandler struct{ service clientService.ClientService }

func NewClientHandler(service clientService.ClientService) ClientHandler {
	return &clientHandler{service: service}
}

// Me godoc
// @Summary Get current client profile
// @Tags Client
// @Produce json
// @Security BearerAuth
// @Success 200 {object} client.MeResponse
// @Failure 401 {object} common.ErrorResponse
// @Router /clients/me [get]
func (h *clientHandler) Me(c *gin.Context) {
	clientID := c.MustGet(middleware.ClientIDKey).(uuid.UUID)
	data, err := h.service.GetMe(c.Request.Context(), clientID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found", nil)
		return
	}
	response.Success(c, http.StatusOK, "Client retrieved successfully", data)
}
