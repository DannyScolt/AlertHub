package alert

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "alerthub/core/domain/alert"
	escalationRepo "alerthub/core/repository/escalation"

	"github.com/google/uuid"
)

type escalationLookupStub struct {
	alert domain.Alert
	err   error
	calls int
}

func (s *escalationLookupStub) FindByID(ctx context.Context, alertID uuid.UUID) (domain.Alert, error) {
	s.calls++
	return s.alert, s.err
}

type escalationWindowStub struct {
	count     int
	sourceIDs []uuid.UUID
	err       error
	since     time.Time
	calls     int
}

func (s *escalationWindowStub) CountSameTypeWithinWindow(ctx context.Context, deviceID uuid.UUID, alertType string, since time.Time) (int, error) {
	s.calls++
	s.since = since
	return s.count, s.err
}

func (s *escalationWindowStub) ListSameTypeIDsWithinWindow(ctx context.Context, deviceID uuid.UUID, alertType string, since time.Time) ([]uuid.UUID, error) {
	s.calls++
	s.since = since
	if s.sourceIDs != nil {
		return s.sourceIDs, s.err
	}
	ids := make([]uuid.UUID, 0, s.count)
	for range s.count {
		ids = append(ids, uuid.New())
	}
	return ids, s.err
}

type escalationIngestStub struct {
	alerts []domain.Alert
	err    error
}

func (s *escalationIngestStub) Create(ctx context.Context, alert domain.Alert) (domain.Alert, error) {
	s.alerts = append(s.alerts, alert)
	alert.ID = uuid.New()
	return alert, s.err
}

func (s *escalationIngestStub) CreateBatch(ctx context.Context, alerts []domain.Alert) ([]domain.Alert, error) {
	return nil, errors.New("not used")
}

type escalationNotifierStub struct {
	calls int
	err   error
}

func (s *escalationNotifierStub) NotifyAlertCreated(ctx context.Context, clientID, alertID uuid.UUID) error {
	s.calls++
	return s.err
}

type escalationCooldownStub struct {
	claimed bool
	err     error
	calls   int
}

func (s *escalationCooldownStub) ClaimEscalation(ctx context.Context, key escalationRepo.CooldownKey, ttl time.Duration) (bool, error) {
	s.calls++
	return s.claimed, s.err
}

func TestHandleNewAlert_DisabledConfigSkipsAllRepos(t *testing.T) {
	deps := newEscalationDeps()
	service := NewEscalationService(deps.lookup, deps.window, deps.ingest, deps.notifier, deps.cooldown, EscalationConfig{Enabled: false}, deps.now)

	out, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: uuid.New()})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Reason != "disabled" {
		t.Fatalf("expected disabled reason, got %q", out.Reason)
	}
	if deps.lookup.calls != 0 || deps.window.calls != 0 || deps.cooldown.calls != 0 || len(deps.ingest.alerts) != 0 {
		t.Fatal("expected disabled service to skip all repositories")
	}
}

func TestHandleNewAlert_SkipsAutoEscalatedMarker(t *testing.T) {
	deps := newEscalationDeps()
	deps.lookup.alert.Type = domain.TypeAutoEscalated
	service := deps.service()

	out, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Reason != "skipped_marker" {
		t.Fatalf("expected skipped_marker, got %q", out.Reason)
	}
	if deps.window.calls != 0 || deps.cooldown.calls != 0 || len(deps.ingest.alerts) != 0 {
		t.Fatal("expected marker alert to skip counting and emit")
	}
}

func TestHandleNewAlert_BelowThresholdDoesNotEmit(t *testing.T) {
	deps := newEscalationDeps()
	deps.window.count = 2
	service := deps.service()

	out, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Reason != "below_threshold" {
		t.Fatalf("expected below_threshold, got %q", out.Reason)
	}
	if deps.cooldown.calls != 0 || len(deps.ingest.alerts) != 0 {
		t.Fatal("expected below threshold to skip cooldown and emit")
	}
}

func TestHandleNewAlert_AtThresholdEmitsCriticalAlert(t *testing.T) {
	deps := newEscalationDeps()
	deps.window.count = 3
	service := deps.service()

	out, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Reason != "emitted" || !out.Emitted {
		t.Fatalf("expected emitted outcome, got %+v", out)
	}
	if len(deps.ingest.alerts) != 1 {
		t.Fatalf("expected one inserted alert, got %d", len(deps.ingest.alerts))
	}
	emitted := deps.ingest.alerts[0]
	if emitted.Severity != domain.SeverityCritical || emitted.Type != domain.TypeAutoEscalated {
		t.Fatalf("unexpected emitted alert severity/type: %s/%s", emitted.Severity, emitted.Type)
	}
	if emitted.DeviceID != deps.lookup.alert.DeviceID || emitted.ClientID != deps.lookup.alert.ClientID {
		t.Fatal("expected emitted alert to keep device and client")
	}
	if deps.notifier.calls != 1 {
		t.Fatalf("expected notifier once, got %d", deps.notifier.calls)
	}
}

func TestHandleNewAlert_AboveThresholdEmitsExactlyOnce(t *testing.T) {
	deps := newEscalationDeps()
	deps.window.count = 5
	service := deps.service()

	_, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(deps.ingest.alerts) != 1 || deps.cooldown.calls != 1 {
		t.Fatalf("expected exactly one emit and cooldown claim, got emits=%d claims=%d", len(deps.ingest.alerts), deps.cooldown.calls)
	}
}

func TestHandleNewAlert_InCooldownDoesNotEmit(t *testing.T) {
	deps := newEscalationDeps()
	deps.window.count = 3
	deps.cooldown.claimed = false
	service := deps.service()

	out, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Reason != "cooldown_active" {
		t.Fatalf("expected cooldown_active, got %q", out.Reason)
	}
	if len(deps.ingest.alerts) != 0 {
		t.Fatal("expected no insert during cooldown")
	}
}

func TestHandleNewAlert_PayloadShapeIncludesAllSourceIDsCountWindowThreshold(t *testing.T) {
	deps := newEscalationDeps()
	sourceIDs := []uuid.UUID{uuid.New(), deps.lookup.alert.ID, uuid.New()}
	deps.window.count = len(sourceIDs)
	deps.window.sourceIDs = sourceIDs
	service := deps.service()

	_, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	payload := deps.ingest.alerts[0].Payload
	if payload["count"] != 3 || payload["window_seconds"] != 60 || payload["threshold"] != 3 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	actual, ok := payload["source_alert_ids"].([]string)
	if !ok {
		t.Fatalf("unexpected source ids type: %#v", payload["source_alert_ids"])
	}
	expected := []string{sourceIDs[0].String(), sourceIDs[1].String(), sourceIDs[2].String()}
	if len(actual) != len(expected) {
		t.Fatalf("expected %d source ids, got %#v", len(expected), actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected source ids %#v, got %#v", expected, actual)
		}
	}
	if payload["detected_at"] != deps.now() {
		t.Fatalf("unexpected detected_at: %#v", payload["detected_at"])
	}
}

func TestHandleNewAlert_LookupErrorPropagates(t *testing.T) {
	deps := newEscalationDeps()
	deps.lookup.err = errors.New("lookup failed")
	service := deps.service()

	_, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: uuid.New()})
	if err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestHandleNewAlert_WindowQueryUsesNowMinusWindow(t *testing.T) {
	deps := newEscalationDeps()
	deps.window.count = 2
	service := deps.service()

	_, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := deps.now().Add(-time.Minute)
	if !deps.window.since.Equal(expected) {
		t.Fatalf("expected since %s, got %s", expected, deps.window.since)
	}
}

func TestHandleNewAlert_NotifierFailureSurfacesAfterInsert(t *testing.T) {
	deps := newEscalationDeps()
	deps.window.count = 3
	deps.notifier.err = errors.New("notify failed")
	service := deps.service()

	_, err := service.HandleNewAlert(context.Background(), EscalateInput{AlertID: deps.lookup.alert.ID})
	if err == nil {
		t.Fatal("expected notifier error")
	}
	if len(deps.ingest.alerts) != 1 {
		t.Fatalf("expected insert before notifier failure, got %d", len(deps.ingest.alerts))
	}
}

type escalationDeps struct {
	lookup   *escalationLookupStub
	window   *escalationWindowStub
	ingest   *escalationIngestStub
	notifier *escalationNotifierStub
	cooldown *escalationCooldownStub
	now      func() time.Time
}

func newEscalationDeps() escalationDeps {
	detectedAt := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	alert := domain.Alert{ID: uuid.New(), DeviceID: uuid.New(), ClientID: uuid.New(), Type: "temperature", Severity: domain.SeverityWarning, Message: "hot", OccurredAt: detectedAt}
	return escalationDeps{
		lookup:   &escalationLookupStub{alert: alert},
		window:   &escalationWindowStub{count: 0},
		ingest:   &escalationIngestStub{},
		notifier: &escalationNotifierStub{},
		cooldown: &escalationCooldownStub{claimed: true},
		now:      func() time.Time { return detectedAt },
	}
}

func (d escalationDeps) service() EscalationService {
	return NewEscalationService(d.lookup, d.window, d.ingest, d.notifier, d.cooldown, EscalationConfig{Enabled: true, Threshold: 3, Window: time.Minute, Cooldown: 5 * time.Minute}, d.now)
}
