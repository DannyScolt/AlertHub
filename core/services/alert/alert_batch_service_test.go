package alert

import (
	"context"
	"testing"
	"time"

	domain "alerthub/core/domain/alert"
	alertDto "alerthub/core/dto/alert"

	"github.com/google/uuid"
)

type batchAlertRepoStub struct {
	batch []domain.Alert
}

func (r *batchAlertRepoStub) Create(ctx context.Context, alert domain.Alert) (domain.Alert, error) {
	return domain.Alert{}, nil
}

func (r *batchAlertRepoStub) CreateBatch(ctx context.Context, alerts []domain.Alert) ([]domain.Alert, error) {
	r.batch = append([]domain.Alert{}, alerts...)
	saved := make([]domain.Alert, 0, len(alerts))
	for _, alert := range alerts {
		alert.ID = uuid.New()
		alert.ReceivedAt = time.Now().UTC()
		alert.CreatedAt = alert.ReceivedAt
		saved = append(saved, alert)
	}
	return saved, nil
}

func (r *batchAlertRepoStub) FindByID(ctx context.Context, alertID uuid.UUID) (domain.Alert, error) {
	return domain.Alert{}, nil
}

func (r *batchAlertRepoStub) LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error) {
	return nil, nil
}

func TestIngestBatchAcceptsAllValidEvents(t *testing.T) {
	repo := &batchAlertRepoStub{}
	notifier := &ingestNotifierStub{}
	service := NewIngestService(repo, notifier)

	resp, err := service.IngestBatch(context.Background(), uuid.New(), uuid.New(), alertDto.BatchRequest{Events: []alertDto.IngestRequest{
		{Type: "temperature", Severity: "warning", Message: "Temperature high"},
		{Type: "smoke", Severity: "critical", Message: "Smoke detected"},
	}})
	if err != nil {
		t.Fatalf("IngestBatch returned error: %v", err)
	}
	if resp.Accepted != 2 || resp.Rejected != 0 || len(resp.Alerts) != 2 || len(resp.Errors) != 0 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(repo.batch) != 2 {
		t.Fatalf("expected two inserted alerts, got %d", len(repo.batch))
	}
	if notifier.calls != 2 {
		t.Fatalf("expected two notifications, got %d", notifier.calls)
	}
}

func TestIngestBatchReturnsPartialFailures(t *testing.T) {
	repo := &batchAlertRepoStub{}
	service := NewIngestService(repo, &ingestNotifierStub{})

	resp, err := service.IngestBatch(context.Background(), uuid.New(), uuid.New(), alertDto.BatchRequest{Events: []alertDto.IngestRequest{
		{Type: "temperature", Severity: "warning", Message: "Temperature high"},
		{Type: "temperature", Severity: "bad", Message: "Temperature high"},
	}})
	if err != nil {
		t.Fatalf("IngestBatch returned error: %v", err)
	}
	if resp.Accepted != 1 || resp.Rejected != 1 || len(resp.Alerts) != 1 || len(resp.Errors) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Errors[0].Index != 1 || resp.Errors[0].Code == "" {
		t.Fatalf("unexpected batch error: %+v", resp.Errors[0])
	}
	if len(repo.batch) != 1 {
		t.Fatalf("expected only valid event inserted, got %d", len(repo.batch))
	}
}

func TestIngestBatchRejectsEmptyArray(t *testing.T) {
	service := NewIngestService(&batchAlertRepoStub{}, &ingestNotifierStub{})

	_, err := service.IngestBatch(context.Background(), uuid.New(), uuid.New(), alertDto.BatchRequest{})
	if err == nil {
		t.Fatal("expected error for empty batch")
	}
}

func TestIngestBatchRejectsMoreThanMaxEvents(t *testing.T) {
	service := NewIngestService(&batchAlertRepoStub{}, &ingestNotifierStub{})
	events := make([]alertDto.IngestRequest, alertDto.BatchMaxSize+1)
	for i := range events {
		events[i] = alertDto.IngestRequest{Type: "temperature", Severity: "info", Message: "Temperature sample"}
	}

	_, err := service.IngestBatch(context.Background(), uuid.New(), uuid.New(), alertDto.BatchRequest{Events: events})
	if err == nil {
		t.Fatal("expected error for oversized batch")
	}
}
