package alert

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "alerthub/core/domain/alert"
	alertDto "alerthub/core/dto/alert"
	alertRepo "alerthub/core/repository/alert"

	"github.com/google/uuid"
)

var (
	ErrInvalidSeverity = errors.New("invalid alert severity")
	ErrInvalidType     = errors.New("invalid alert type")
	ErrInvalidMessage  = errors.New("invalid alert message")
	ErrEmptyBatch      = errors.New("batch events must not be empty")
	ErrBatchTooLarge   = errors.New("batch events exceeds max size")
)

type IngestService interface {
	IngestEvent(ctx context.Context, deviceID, clientID uuid.UUID, req alertDto.IngestRequest) (alertDto.IngestResponse, error)
	IngestBatch(ctx context.Context, deviceID, clientID uuid.UUID, req alertDto.BatchRequest) (alertDto.BatchResponse, error)
}

type ingestService struct {
	repo     alertRepo.IngestRepository
	notifier alertRepo.Notifier
}

func NewIngestService(repo alertRepo.IngestRepository, notifier alertRepo.Notifier) IngestService {
	return &ingestService{repo: repo, notifier: notifier}
}

func (s *ingestService) IngestEvent(ctx context.Context, deviceID, clientID uuid.UUID, req alertDto.IngestRequest) (alertDto.IngestResponse, error) {
	alert, err := buildAlert(deviceID, clientID, req)
	if err != nil {
		return alertDto.IngestResponse{}, err
	}
	saved, err := s.repo.Create(ctx, alert)
	if err != nil {
		return alertDto.IngestResponse{}, err
	}
	if err := s.notifier.NotifyAlertCreated(ctx, saved.ClientID, saved.ID); err != nil {
		return alertDto.IngestResponse{}, err
	}
	return alertDto.IngestResponse{AlertID: saved.ID.String(), ReceivedAt: saved.ReceivedAt}, nil
}

func (s *ingestService) IngestBatch(ctx context.Context, deviceID, clientID uuid.UUID, req alertDto.BatchRequest) (alertDto.BatchResponse, error) {
	if len(req.Events) == 0 {
		return alertDto.BatchResponse{}, ErrEmptyBatch
	}
	if len(req.Events) > alertDto.BatchMaxSize {
		return alertDto.BatchResponse{}, ErrBatchTooLarge
	}

	validAlerts := make([]domain.Alert, 0, len(req.Events))
	indexes := make([]int, 0, len(req.Events))
	errors := make([]alertDto.BatchError, 0)
	for i, event := range req.Events {
		alert, err := buildAlert(deviceID, clientID, event)
		if err != nil {
			errors = append(errors, alertDto.BatchError{Index: i, Code: batchErrorCode(err), Message: err.Error()})
			continue
		}
		validAlerts = append(validAlerts, alert)
		indexes = append(indexes, i)
	}

	savedAlerts, err := s.repo.CreateBatch(ctx, validAlerts)
	if err != nil {
		return alertDto.BatchResponse{}, err
	}
	accepted := make([]alertDto.BatchAcceptedAlert, 0, len(savedAlerts))
	for i, saved := range savedAlerts {
		if err := s.notifier.NotifyAlertCreated(ctx, saved.ClientID, saved.ID); err != nil {
			return alertDto.BatchResponse{}, err
		}
		accepted = append(accepted, alertDto.BatchAcceptedAlert{Index: indexes[i], AlertID: saved.ID.String()})
	}

	return alertDto.BatchResponse{Accepted: len(accepted), Rejected: len(errors), Alerts: accepted, Errors: errors}, nil
}

func buildAlert(deviceID, clientID uuid.UUID, req alertDto.IngestRequest) (domain.Alert, error) {
	severity := domain.Severity(req.Severity)
	if !domain.ValidSeverity(severity) {
		return domain.Alert{}, ErrInvalidSeverity
	}
	if !domain.ValidateType(req.Type) {
		return domain.Alert{}, ErrInvalidType
	}
	if !domain.ValidateMessage(req.Message) {
		return domain.Alert{}, ErrInvalidMessage
	}
	occurredAt := time.Now().UTC()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}
	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	return domain.Alert{DeviceID: deviceID, ClientID: clientID, Type: strings.TrimSpace(req.Type), Severity: severity, Message: strings.TrimSpace(req.Message), Payload: payload, OccurredAt: occurredAt}, nil
}

func batchErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidSeverity):
		return "INVALID_SEVERITY"
	case errors.Is(err, ErrInvalidType):
		return "INVALID_TYPE"
	case errors.Is(err, ErrInvalidMessage):
		return "INVALID_MESSAGE"
	default:
		return "INVALID_EVENT"
	}
}
