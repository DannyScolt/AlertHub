package alert

import (
	"context"
	"fmt"
	"time"

	domain "alerthub/core/domain/alert"
	alertDto "alerthub/core/dto/alert"
	alertRepo "alerthub/core/repository/alert"
	escalationRepo "alerthub/core/repository/escalation"

	"github.com/google/uuid"
)

type EscalationConfig struct {
	Enabled   bool
	Threshold int
	Window    time.Duration
	Cooldown  time.Duration
}

type EscalateInput struct {
	AlertID uuid.UUID
}

type EscalationOutcome struct {
	Emitted bool
	Reason  string
}

type EscalationService interface {
	HandleNewAlert(ctx context.Context, input EscalateInput) (EscalationOutcome, error)
}

type escalationService struct {
	lookup   alertRepo.LookupRepository
	window   alertRepo.WindowCounter
	ingest   alertRepo.IngestRepository
	notifier alertRepo.Notifier
	cooldown escalationRepo.CooldownStore
	cfg      EscalationConfig
	now      func() time.Time
}

func NewEscalationService(
	lookup alertRepo.LookupRepository,
	window alertRepo.WindowCounter,
	ingest alertRepo.IngestRepository,
	notifier alertRepo.Notifier,
	cooldown escalationRepo.CooldownStore,
	cfg EscalationConfig,
	now func() time.Time,
) EscalationService {
	return &escalationService{lookup: lookup, window: window, ingest: ingest, notifier: notifier, cooldown: cooldown, cfg: cfg, now: now}
}

func (s *escalationService) HandleNewAlert(ctx context.Context, input EscalateInput) (EscalationOutcome, error) {
	if !s.cfg.Enabled {
		return EscalationOutcome{Reason: "disabled"}, nil
	}

	source, err := s.lookup.FindByID(ctx, input.AlertID)
	if err != nil {
		return EscalationOutcome{}, fmt.Errorf("lookup alert for escalation: %w", err)
	}
	if source.Type == domain.TypeAutoEscalated {
		return EscalationOutcome{Reason: "skipped_marker"}, nil
	}

	sourceIDs, err := s.window.ListSameTypeIDsWithinWindow(ctx, source.DeviceID, source.Type, s.now().Add(-s.cfg.Window))
	if err != nil {
		return EscalationOutcome{}, fmt.Errorf("list escalation window sources: %w", err)
	}
	if len(sourceIDs) < s.cfg.Threshold {
		return EscalationOutcome{Reason: "below_threshold"}, nil
	}

	claimed, err := s.cooldown.ClaimEscalation(ctx, escalationRepo.CooldownKey{DeviceID: source.DeviceID, AlertType: source.Type}, s.cfg.Cooldown)
	if err != nil {
		return EscalationOutcome{}, fmt.Errorf("claim escalation cooldown: %w", err)
	}
	if !claimed {
		return EscalationOutcome{Reason: "cooldown_active"}, nil
	}

	escalated, err := s.ingest.Create(ctx, s.buildEscalation(source, sourceIDs))
	if err != nil {
		return EscalationOutcome{}, fmt.Errorf("insert escalation alert: %w", err)
	}
	if err := s.notifier.NotifyAlertCreated(ctx, escalated.ClientID, escalated.ID); err != nil {
		return EscalationOutcome{}, fmt.Errorf("notify escalation alert: %w", err)
	}
	return EscalationOutcome{Emitted: true, Reason: "emitted"}, nil
}

func stringifyUUIDs(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func (s *escalationService) buildEscalation(source domain.Alert, sourceIDs []uuid.UUID) domain.Alert {
	detectedAt := s.now()
	payload := alertDto.EscalationPayload{
		SourceAlertIDs: stringifyUUIDs(sourceIDs),
		Count:          len(sourceIDs),
		WindowSeconds:  int(s.cfg.Window.Seconds()),
		Threshold:      s.cfg.Threshold,
		DetectedAt:     detectedAt,
	}
	return domain.Alert{
		DeviceID:   source.DeviceID,
		ClientID:   source.ClientID,
		Type:       domain.TypeAutoEscalated,
		Severity:   domain.SeverityCritical,
		Message:    "Repeated alert burst auto-escalated",
		Payload:    map[string]interface{}{"source_alert_ids": payload.SourceAlertIDs, "count": payload.Count, "window_seconds": payload.WindowSeconds, "threshold": payload.Threshold, "detected_at": payload.DetectedAt},
		OccurredAt: detectedAt,
	}
}
