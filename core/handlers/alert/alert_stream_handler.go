package alert

import (
	"net/http"
	"time"

	alertDto "alerthub/core/dto/alert"
	"alerthub/core/middleware"
	alertService "alerthub/core/services/alert"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const heartbeatInterval = 30 * time.Second

type StreamHandler interface {
	Stream(*gin.Context)
}

type streamHandler struct{ service alertService.StreamService }

func NewStreamHandler(service alertService.StreamService) StreamHandler {
	return &streamHandler{service: service}
}

// Stream godoc
// @Summary Stream realtime alerts to an authenticated client with Server-Sent Events
// @Description Opens an SSE connection for the authenticated client. Use client JWT auth (`Authorization: Bearer <access_token>`), not a device API key. The stream immediately sends `event: connected`, then sends `event: alert` whenever one of the client's devices ingests an alert, and sends `event: heartbeat` every 30 seconds to keep the connection alive. Optional `device_id` filters alerts to one device owned by the client. Events from other clients are never delivered.
// @Tags Alerts
// @Produce text/event-stream
// @Security BearerAuth
// @Param device_id query string false "Optional device UUID filter. When set, only alerts for this device are sent." example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Success 200 {object} alert.StreamAlertEvent "SSE stream. Event types: connected, alert, heartbeat."
// @Failure 400 {object} common.ErrorResponse "Invalid device_id query parameter."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid client access token."
// @Router /alerts/stream [get]
func (h *streamHandler) Stream(c *gin.Context) {
	var deviceID *uuid.UUID
	if rawDeviceID := c.Query("device_id"); rawDeviceID != "" {
		parsed, err := uuid.Parse(rawDeviceID)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid device_id", nil)
			return
		}
		deviceID = &parsed
	}

	clientID := c.MustGet(middleware.ClientIDKey).(uuid.UUID)
	subscriber := h.service.Subscribe(clientID, deviceID)
	defer h.service.Unsubscribe(subscriber.ID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.SSEvent("connected", alertDto.StreamConnectedEvent{ClientID: clientID.String(), Timestamp: time.Now().UTC()})
	c.Writer.Flush()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case alert := <-subscriber.Alerts:
			c.SSEvent("alert", alertDto.StreamAlertEvent{ID: alert.ID.String(), DeviceID: alert.DeviceID.String(), Type: alert.Type, Severity: string(alert.Severity), Message: alert.Message, Payload: alert.Payload, OccurredAt: alert.OccurredAt, ReceivedAt: alert.ReceivedAt})
			c.Writer.Flush()
		case <-ticker.C:
			c.SSEvent("heartbeat", alertDto.StreamHeartbeatEvent{Timestamp: time.Now().UTC()})
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
