package alert

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "alerthub/core/domain/alert"
	alertDto "alerthub/core/dto/alert"

	"github.com/google/uuid"
)

type ingestAlertRepoStub struct {
	created []domain.Alert
	next    domain.Alert
}

func (r *ingestAlertRepoStub) Create(ctx context.Context, alert domain.Alert) (domain.Alert, error) {
	r.created = append(r.created, alert)
	if r.next.ID == uuid.Nil {
		r.next = alert
		r.next.ID = uuid.New()
		r.next.ReceivedAt = time.Now()
		r.next.CreatedAt = r.next.ReceivedAt
	}
	return r.next, nil
}

func (r *ingestAlertRepoStub) CreateBatch(ctx context.Context, alerts []domain.Alert) ([]domain.Alert, error) {
	return nil, nil
}

func (r *ingestAlertRepoStub) FindByID(ctx context.Context, alertID uuid.UUID) (domain.Alert, error) {
	return domain.Alert{}, nil
}

func (r *ingestAlertRepoStub) LatestOccurredAtByDeviceID(ctx context.Context, deviceID uuid.UUID) (*time.Time, error) {
	return nil, nil
}

type ingestNotifierStub struct {
	calls    int
	clientID uuid.UUID
	alertID  uuid.UUID
	err      error
}

func (n *ingestNotifierStub) NotifyAlertCreated(ctx context.Context, clientID, alertID uuid.UUID) error {
	n.calls++
	n.clientID = clientID
	n.alertID = alertID
	return n.err
}

func TestIngestEventCreatesAlertAndNotifies(t *testing.T) {
	deviceID := uuid.New()
	clientID := uuid.New()
	occurredAt := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	repo := &ingestAlertRepoStub{}
	notifier := &ingestNotifierStub{}
	service := NewIngestService(repo, notifier)

	resp, err := service.IngestEvent(context.Background(), deviceID, clientID, alertDto.IngestRequest{
		Type:       "high_temperature",
		Severity:   "warning",
		Message:    "Temperature exceeded 80°C",
		Payload:    map[string]interface{}{"value": 82.5},
		OccurredAt: &occurredAt,
	})
	if err != nil {
		t.Fatalf("IngestEvent returned error: %v", err)
	}
	if resp.AlertID == "" {
		t.Fatal("expected alert id in response")
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected one created alert, got %d", len(repo.created))
	}
	created := repo.created[0]
	if created.DeviceID != deviceID || created.ClientID != clientID {
		t.Fatal("created alert has wrong ownership")
	}
	if created.Type != "high_temperature" || created.Severity != domain.SeverityWarning || created.Message != "Temperature exceeded 80°C" {
		t.Fatalf("created alert has wrong fields: %+v", created)
	}
	if !created.OccurredAt.Equal(occurredAt) {
		t.Fatalf("expected occurred_at %s, got %s", occurredAt, created.OccurredAt)
	}
	if notifier.calls != 1 || notifier.clientID != clientID || notifier.alertID.String() != resp.AlertID {
		t.Fatalf("notify mismatch: calls=%d client=%s alert=%s resp=%s", notifier.calls, notifier.clientID, notifier.alertID, resp.AlertID)
	}
}

func TestIngestEventDefaultsOccurredAtAndPayload(t *testing.T) {
	repo := &ingestAlertRepoStub{}
	service := NewIngestService(repo, &ingestNotifierStub{})

	_, err := service.IngestEvent(context.Background(), uuid.New(), uuid.New(), alertDto.IngestRequest{
		Type:     "door_opened",
		Severity: "info",
		Message:  "Door opened",
	})
	if err != nil {
		t.Fatalf("IngestEvent returned error: %v", err)
	}
	created := repo.created[0]
	if created.OccurredAt.IsZero() {
		t.Fatal("expected default occurred_at")
	}
	if created.Payload == nil {
		t.Fatal("expected default payload map")
	}
}

func TestIngestEventRejectsInvalidInput(t *testing.T) {
	service := NewIngestService(&ingestAlertRepoStub{}, &ingestNotifierStub{})
	tests := []alertDto.IngestRequest{
		{Type: "", Severity: "info", Message: "message"},
		{Type: "event", Severity: "bad", Message: "message"},
		{Type: "event", Severity: "info", Message: ""},
	}
	for _, req := range tests {
		if _, err := service.IngestEvent(context.Background(), uuid.New(), uuid.New(), req); err == nil {
			t.Fatalf("expected error for request %+v", req)
		}
	}
}

func TestIngestEventReturnsErrorWhenNotifyFails(t *testing.T) {
	repo := &ingestAlertRepoStub{}
	service := NewIngestService(repo, &ingestNotifierStub{err: errors.New("notify failed")})

	_, err := service.IngestEvent(context.Background(), uuid.New(), uuid.New(), alertDto.IngestRequest{
		Type:     "high_temperature",
		Severity: "critical",
		Message:  "Temperature exceeded 100°C",
	})
	if err == nil {
		t.Fatal("expected notify error")
	}
}
