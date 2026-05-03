package alert

import (
	"errors"
	"net/http"

	alertDto "alerthub/core/dto/alert"
	"alerthub/core/middleware"
	alertService "alerthub/core/services/alert"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IngestHandler interface {
	Ingest(*gin.Context)
	IngestBatch(*gin.Context)
}

type ingestHandler struct{ service alertService.IngestService }

func NewIngestHandler(service alertService.IngestService) IngestHandler {
	return &ingestHandler{service: service}
}

// Ingest godoc
// @Summary Ingest one realtime alert event from a device
// @Description Accepts one append-only alert event from an authenticated device. Authenticate with the raw device API key returned by POST /devices or POST /devices/{id}/rotate-key using `Authorization: Bearer ah_dev_...`. The API stores only the alert row, emits a PostgreSQL NOTIFY event for SSE subscribers, and never exposes api_key/api_key_hash in the response. `severity` must be one of info, warning, critical; `type` is a free string up to 100 characters; `payload` is optional JSON metadata; `occurred_at` is optional and defaults to server time.
// @Tags Alerts
// @Accept json
// @Produce json
// @Security DeviceAPIKey
// @Param request body alert.IngestRequest true "Single alert payload. Example severity values: info, warning, critical."
// @Success 202 {object} alert.IngestEnvelopeResponse "Event accepted; data.alert_id can be used to correlate stream events."
// @Failure 400 {object} common.ErrorResponse "Validation error: invalid severity, blank type, blank message, malformed JSON."
// @Failure 401 {object} common.ErrorResponse "Missing, invalid, or soft-deleted device API key."
// @Failure 500 {object} common.ErrorResponse "Internal insert or notify error."
// @Router /events [post]
func (h *ingestHandler) Ingest(c *gin.Context) {
	var req alertDto.IngestRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.IngestEvent(c.Request.Context(), currentDeviceID(c), currentClientID(c), req)
	if err != nil {
		handleAlertError(c, err)
		return
	}
	response.Success(c, http.StatusAccepted, "Event accepted", data)
}

// IngestBatch godoc
// @Summary Ingest a batch of realtime alert events from a device
// @Description Accepts up to 100 alert events from one authenticated device. Valid events are inserted and notified; invalid events are returned in data.errors with their original index, so clients can retry only failed items. The response can contain both accepted alerts and rejected errors. Authentication uses `Authorization: Bearer ah_dev_...` with the raw device API key.
// @Tags Alerts
// @Accept json
// @Produce json
// @Security DeviceAPIKey
// @Param request body alert.BatchRequest true "Batch payload. events must contain 1..100 items. Partial failure is reported per index."
// @Success 202 {object} alert.BatchEnvelopeResponse "Batch processed; check data.accepted, data.rejected, data.alerts, and data.errors."
// @Failure 400 {object} common.ErrorResponse "Empty batch, more than 100 events, malformed JSON, or invalid batch shape."
// @Failure 401 {object} common.ErrorResponse "Missing, invalid, or soft-deleted device API key."
// @Failure 500 {object} common.ErrorResponse "Internal insert or notify error."
// @Router /events/batch [post]
func (h *ingestHandler) IngestBatch(c *gin.Context) {
	var req alertDto.BatchRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.IngestBatch(c.Request.Context(), currentDeviceID(c), currentClientID(c), req)
	if err != nil {
		handleAlertError(c, err)
		return
	}
	response.Success(c, http.StatusAccepted, "Batch processed", data)
}

func currentDeviceID(c *gin.Context) uuid.UUID {
	return c.MustGet(middleware.DeviceIDKey).(uuid.UUID)
}

func currentClientID(c *gin.Context) uuid.UUID {
	return c.MustGet(middleware.ClientIDKey).(uuid.UUID)
}

func bind(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", err.Error())
		return false
	}
	return true
}

func handleAlertError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, alertService.ErrInvalidSeverity):
		response.Error(c, http.StatusBadRequest, "INVALID_SEVERITY", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidType):
		response.Error(c, http.StatusBadRequest, "INVALID_TYPE", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidMessage):
		response.Error(c, http.StatusBadRequest, "INVALID_MESSAGE", err.Error(), nil)
	case errors.Is(err, alertService.ErrEmptyBatch), errors.Is(err, alertService.ErrBatchTooLarge):
		response.Error(c, http.StatusBadRequest, "INVALID_BATCH", err.Error(), nil)
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
