package alert

import (
	"errors"
	"net/http"
	"strconv"

	domain "alerthub/core/domain/alert"
	alertDto "alerthub/core/dto/alert"
	alertService "alerthub/core/services/alert"
	"alerthub/core/utils/pagination"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
)

type QueryHandler interface {
	List(*gin.Context)
}

type queryHandler struct{ service alertService.QueryService }

func NewQueryHandler(service alertService.QueryService) QueryHandler {
	return &queryHandler{service: service}
}

// List godoc
// @Summary List alerts for the authenticated client
// @Description Returns paginated alerts owned by the authenticated client. Filter by `device_id`, one or more `severity` values (`info`, `warning`, `critical`), a time range on `occurred_at`, and `search` over alert message, alert type, device name, or exact device UUID. Results are scoped to the current client and ordered by `occurred_at DESC, id DESC`. The response never includes `client_id` or any device API key material.
// @Tags Alerts
// @Produce json
// @Security BearerAuth
// @Param device_id query string false "Filter to a single device id (UUID)" example(4d285f4b-2a87-4a86-a5b8-05b09c6d1234)
// @Param severity query []string false "Filter by severity. Repeat to combine, e.g. severity=warning&severity=critical" Enums(info,warning,critical) collectionFormat(multi)
// @Param from query string false "Inclusive lower bound on occurred_at, RFC3339" example(2026-05-01T00:00:00Z)
// @Param to query string false "Inclusive upper bound on occurred_at, RFC3339" example(2026-05-04T23:59:59Z)
// @Param search query string false "Search alert message, alert type, device name, or exact device UUID" minlength(2) maxlength(100) example(smoke)
// @Param page query int false "Page number, starts at 1" minimum(1) default(1)
// @Param page_size query int false "Items per page" minimum(1) maximum(100) default(20)
// @Success 200 {object} alert.ListAlertsResponse "Alerts retrieved successfully with pagination metadata."
// @Failure 400 {object} common.ErrorResponse "Validation error on filters or pagination."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
// @Router /alerts [get]
func (h *queryHandler) List(c *gin.Context) {
	input, err := listAlertsInput(c)
	if err != nil {
		handleAlertQueryError(c, err)
		return
	}

	out, err := h.service.ListAlerts(c.Request.Context(), currentClientID(c), input)
	if err != nil {
		handleAlertQueryError(c, err)
		return
	}
	response.Paginated(c, "Alerts retrieved successfully", toAlertResponses(out.Alerts), pagination.Meta(out.Page, out.PageSize, out.Total))
}

func listAlertsInput(c *gin.Context) (alertService.ListAlertsInput, error) {
	page, err := queryInt(c, "page", pagination.DefaultPage)
	if err != nil {
		return alertService.ListAlertsInput{}, err
	}
	pageSize, err := queryInt(c, "page_size", pagination.DefaultPageSize)
	if err != nil {
		return alertService.ListAlertsInput{}, err
	}
	return alertService.ListAlertsInput{
		DeviceID:   optionalQuery(c, "device_id"),
		Severities: c.QueryArray("severity"),
		From:       optionalQuery(c, "from"),
		To:         optionalQuery(c, "to"),
		Search:     optionalQuery(c, "search"),
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func optionalQuery(c *gin.Context, key string) *string {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	return &value
}

func queryInt(c *gin.Context, key string, fallback int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, alertService.ErrInvalidPagination
	}
	return value, nil
}

func toAlertResponses(alerts []domain.Alert) []alertDto.AlertResponse {
	out := make([]alertDto.AlertResponse, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, alertDto.AlertResponse{
			ID:         a.ID.String(),
			DeviceID:   a.DeviceID.String(),
			Type:       a.Type,
			Severity:   string(a.Severity),
			Message:    a.Message,
			Payload:    a.Payload,
			OccurredAt: a.OccurredAt,
			ReceivedAt: a.ReceivedAt,
		})
	}
	return out
}

func handleAlertQueryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, alertService.ErrInvalidDeviceID):
		response.Error(c, http.StatusBadRequest, "INVALID_DEVICE_ID", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidSeverity):
		response.Error(c, http.StatusBadRequest, "INVALID_SEVERITY", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidTimeFormat):
		response.Error(c, http.StatusBadRequest, "INVALID_TIME_FORMAT", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidTimeRange):
		response.Error(c, http.StatusBadRequest, "INVALID_TIME_RANGE", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidPagination):
		response.Error(c, http.StatusBadRequest, "INVALID_PAGINATION", err.Error(), nil)
	case errors.Is(err, alertService.ErrInvalidSearch):
		response.Error(c, http.StatusBadRequest, "INVALID_SEARCH", err.Error(), nil)
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
