package alert

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "alerthub/core/domain/alert"
	alertRepo "alerthub/core/repository/alert"
	"alerthub/core/utils/pagination"

	"github.com/google/uuid"
)

var (
	ErrInvalidDeviceID   = errors.New("invalid device id")
	ErrInvalidTimeFormat = errors.New("invalid time format")
	ErrInvalidTimeRange  = errors.New("invalid time range")
	ErrInvalidPagination = errors.New("invalid pagination")
	ErrInvalidSearch     = errors.New("invalid search")
)

type ListAlertsInput struct {
	DeviceID   *string
	Severities []string
	From       *string
	To         *string
	Search     *string
	Page       int
	PageSize   int
}

type ListAlertsOutput struct {
	Alerts   []domain.Alert
	Page     int
	PageSize int
	Total    int64
}

type QueryService interface {
	ListAlerts(ctx context.Context, clientID uuid.UUID, input ListAlertsInput) (ListAlertsOutput, error)
}

type queryService struct {
	repo alertRepo.QueryRepository
}

func NewQueryService(repo alertRepo.QueryRepository) QueryService {
	return &queryService{repo: repo}
}

func (s *queryService) ListAlerts(ctx context.Context, clientID uuid.UUID, input ListAlertsInput) (ListAlertsOutput, error) {
	page, pageSize, err := normalizePagination(input.Page, input.PageSize)
	if err != nil {
		return ListAlertsOutput{}, err
	}

	deviceID, err := parseOptionalDeviceID(input.DeviceID)
	if err != nil {
		return ListAlertsOutput{}, err
	}

	severities, err := parseSeverities(input.Severities)
	if err != nil {
		return ListAlertsOutput{}, err
	}

	from, err := parseOptionalTime(input.From)
	if err != nil {
		return ListAlertsOutput{}, err
	}
	to, err := parseOptionalTime(input.To)
	if err != nil {
		return ListAlertsOutput{}, err
	}
	if from != nil && to != nil && from.After(*to) {
		return ListAlertsOutput{}, ErrInvalidTimeRange
	}

	search, err := normalizeSearch(input.Search)
	if err != nil {
		return ListAlertsOutput{}, err
	}

	result, err := s.repo.List(ctx, clientID, alertRepo.ListFilter{
		DeviceID:   deviceID,
		Severities: severities,
		From:       from,
		To:         to,
		Search:     search,
		Page:       page,
		PageSize:   pageSize,
		Offset:     pagination.Offset(page, pageSize),
	})
	if err != nil {
		return ListAlertsOutput{}, err
	}

	return ListAlertsOutput{
		Alerts:   result.Alerts,
		Page:     page,
		PageSize: pageSize,
		Total:    result.Total,
	}, nil
}

func normalizePagination(page, pageSize int) (int, int, error) {
	if page < 1 {
		return 0, 0, ErrInvalidPagination
	}
	if pageSize < 1 || pageSize > pagination.MaxPageSize {
		return 0, 0, ErrInvalidPagination
	}
	return page, pageSize, nil
}

func parseOptionalDeviceID(raw *string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		return nil, ErrInvalidDeviceID
	}
	return &id, nil
}

func parseSeverities(raw []string) ([]domain.Severity, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	seen := make(map[domain.Severity]struct{}, len(raw))
	out := make([]domain.Severity, 0, len(raw))
	for _, value := range raw {
		if value == "" {
			continue
		}
		s := domain.Severity(value)
		if !domain.ValidSeverity(s) {
			return nil, ErrInvalidSeverity
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, ErrInvalidTimeFormat
	}
	return &t, nil
}

func normalizeSearch(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	search := strings.TrimSpace(*raw)
	if search == "" {
		return nil, nil
	}
	if len(search) < 2 || len(search) > 100 {
		return nil, ErrInvalidSearch
	}
	return &search, nil
}
