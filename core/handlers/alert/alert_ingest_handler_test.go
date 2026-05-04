package alert

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	alertDto "alerthub/core/dto/alert"
	"alerthub/core/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ingestServiceStub struct {
	batch alertDto.BatchResponse
}

func (s ingestServiceStub) IngestEvent(ctx context.Context, deviceID, clientID uuid.UUID, req alertDto.IngestRequest) (alertDto.IngestResponse, error) {
	return alertDto.IngestResponse{}, nil
}

func (s ingestServiceStub) IngestBatch(ctx context.Context, deviceID, clientID uuid.UUID, req alertDto.BatchRequest) (alertDto.BatchResponse, error) {
	return s.batch, nil
}

func performBatchRequest(service ingestServiceStub) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewIngestHandler(service)
	router.POST("/events/batch", func(c *gin.Context) {
		c.Set(middleware.DeviceIDKey, uuid.New())
		c.Set(middleware.ClientIDKey, uuid.New())
		handler.IngestBatch(c)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/events/batch", strings.NewReader(`{"events":[{"type":"temperature_high","severity":"warning","message":"Temperature high"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestBatchIngestReturnsAcceptedForAllValidEvents(t *testing.T) {
	recorder := performBatchRequest(ingestServiceStub{batch: alertDto.BatchResponse{Accepted: 2, Rejected: 0}})

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", recorder.Code)
	}
}

func TestBatchIngestReturnsMultiStatusForPartialFailure(t *testing.T) {
	recorder := performBatchRequest(ingestServiceStub{batch: alertDto.BatchResponse{Accepted: 1, Rejected: 1}})

	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected status 207, got %d", recorder.Code)
	}
}
