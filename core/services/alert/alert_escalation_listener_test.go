package alert

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type escalationServiceStub struct {
	input EscalateInput
	calls int
}

func (s *escalationServiceStub) HandleNewAlert(ctx context.Context, input EscalateInput) (EscalationOutcome, error) {
	s.calls++
	s.input = input
	return EscalationOutcome{}, nil
}

func TestEscalationListenerInvokesServiceForReceivedNotifications(t *testing.T) {
	service := &escalationServiceStub{}
	listener := NewEscalationListener(nil, service)
	alertID := uuid.New()

	listener.handleNotification(context.Background(), &pgconn.Notification{Payload: `{"alert_id":"` + alertID.String() + `"}`})

	if service.calls != 1 {
		t.Fatalf("expected service once, got %d", service.calls)
	}
	if service.input.AlertID != alertID {
		t.Fatalf("expected alert id %s, got %s", alertID, service.input.AlertID)
	}
}
