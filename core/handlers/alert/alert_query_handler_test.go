package alert

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	alertDomain "alerthub/core/domain/alert"
	alertService "alerthub/core/services/alert"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type queryServiceStub struct {
	input alertService.ListAlertsInput
}

func (s *queryServiceStub) ListAlerts(ctx context.Context, clientID uuid.UUID, input alertService.ListAlertsInput) (alertService.ListAlertsOutput, error) {
	s.input = input
	return alertService.ListAlertsOutput{Alerts: []alertDomain.Alert{}, Page: input.Page, PageSize: input.PageSize, Total: 0}, nil
}

func TestListRejectsNonNumericPage(t *testing.T) {
	recorder := performListAlertsRequest("/alerts?page=abc", &queryServiceStub{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestListRejectsNonNumericPageSize(t *testing.T) {
	recorder := performListAlertsRequest("/alerts?page_size=abc", &queryServiceStub{})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func performListAlertsRequest(target string, service alertService.QueryService) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewQueryHandler(service)
	router.GET("/alerts", func(c *gin.Context) {
		c.Set("client_id", uuid.New())
		handler.List(c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}
