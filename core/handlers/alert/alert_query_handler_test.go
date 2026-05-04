package alert

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	alertDomain "alerthub/core/domain/alert"
	alertService "alerthub/core/services/alert"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type queryServiceStub struct {
	input alertService.ListAlertsInput
	err   error
}

func (s *queryServiceStub) ListAlerts(ctx context.Context, clientID uuid.UUID, input alertService.ListAlertsInput) (alertService.ListAlertsOutput, error) {
	s.input = input
	if s.err != nil {
		return alertService.ListAlertsOutput{}, s.err
	}
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

func TestListParsesSearch(t *testing.T) {
	service := &queryServiceStub{}
	recorder := performListAlertsRequest("/alerts?search=smoke", service)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if service.input.Search == nil || *service.input.Search != "smoke" {
		t.Fatalf("expected search %q, got %v", "smoke", service.input.Search)
	}
}

func TestListMapsInvalidSearch(t *testing.T) {
	recorder := performListAlertsRequest("/alerts?search=a", &queryServiceStub{err: alertService.ErrInvalidSearch})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); !containsAll(body, "INVALID_SEARCH", "invalid search") {
		t.Fatalf("expected INVALID_SEARCH response, got %s", body)
	}
}

func containsAll(s string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(s, value) {
			return false
		}
	}
	return true
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
