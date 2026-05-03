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
// @Summary Get the authenticated client profile
// @Description Returns the profile for the client identified by the Bearer access token. This endpoint never returns password hashes, refresh tokens, or client_tokens records.
// @Tags Client
// @Produce json
// @Security BearerAuth
// @Success 200 {object} client.MeResponse "Current client profile."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Failure 404 {object} common.ErrorResponse "Authenticated client no longer exists."
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
