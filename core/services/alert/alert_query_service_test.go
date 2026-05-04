package alert

import (
	"context"
	"errors"
	"testing"

	alertRepo "alerthub/core/repository/alert"

	"github.com/google/uuid"
)

type queryRepoStub struct {
	calledClientID uuid.UUID
	calledFilter   alertRepo.ListFilter
	result         alertRepo.ListResult
	err            error
}

func (r *queryRepoStub) List(ctx context.Context, clientID uuid.UUID, filter alertRepo.ListFilter) (alertRepo.ListResult, error) {
	r.calledClientID = clientID
	r.calledFilter = filter
	return r.result, r.err
}

func ptr(s string) *string { return &s }

func newQueryService(repo alertRepo.QueryRepository) QueryService { return NewQueryService(repo) }

func TestListAlertsRejectsInvalidSeverity(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	_, err := service.ListAlerts(context.Background(), uuid.New(), ListAlertsInput{
		Severities: []string{"warning", "urgent"},
		Page:       1,
		PageSize:   20,
	})

	if !errors.Is(err, ErrInvalidSeverity) {
		t.Fatalf("expected ErrInvalidSeverity, got %v", err)
	}
}

func TestListAlertsRejectsInvalidTimeFormat(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	_, err := service.ListAlerts(context.Background(), uuid.New(), ListAlertsInput{
		From:     ptr("not-a-time"),
		Page:     1,
		PageSize: 20,
	})

	if !errors.Is(err, ErrInvalidTimeFormat) {
		t.Fatalf("expected ErrInvalidTimeFormat, got %v", err)
	}
}

func TestListAlertsRejectsInvertedTimeRange(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	_, err := service.ListAlerts(context.Background(), uuid.New(), ListAlertsInput{
		From:     ptr("2026-05-04T12:00:00Z"),
		To:       ptr("2026-05-04T10:00:00Z"),
		Page:     1,
		PageSize: 20,
	})

	if !errors.Is(err, ErrInvalidTimeRange) {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
}

func TestListAlertsRejectsInvalidPagination(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	cases := []ListAlertsInput{
		{Page: 0, PageSize: 20},
		{Page: 1, PageSize: 0},
		{Page: 1, PageSize: 101},
	}
	for _, input := range cases {
		_, err := service.ListAlerts(context.Background(), uuid.New(), input)
		if !errors.Is(err, ErrInvalidPagination) {
			t.Fatalf("input %+v: expected ErrInvalidPagination, got %v", input, err)
		}
	}
}

func TestListAlertsRejectsInvalidDeviceID(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	_, err := service.ListAlerts(context.Background(), uuid.New(), ListAlertsInput{
		DeviceID: ptr("not-a-uuid"),
		Page:     1,
		PageSize: 20,
	})

	if !errors.Is(err, ErrInvalidDeviceID) {
		t.Fatalf("expected ErrInvalidDeviceID, got %v", err)
	}
}

func TestListAlertsScopesClientIDAndPassesFilter(t *testing.T) {
	clientID := uuid.New()
	deviceID := uuid.New()
	repo := &queryRepoStub{result: alertRepo.ListResult{Total: 0, Alerts: nil}}
	service := newQueryService(repo)

	deviceIDStr := deviceID.String()
	_, err := service.ListAlerts(context.Background(), clientID, ListAlertsInput{
		DeviceID:   &deviceIDStr,
		Severities: []string{"warning", "critical"},
		From:       ptr("2026-05-01T00:00:00Z"),
		To:         ptr("2026-05-04T23:59:59Z"),
		Page:       2,
		PageSize:   25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.calledClientID != clientID {
		t.Fatalf("expected clientID %s, got %s", clientID, repo.calledClientID)
	}
	if repo.calledFilter.DeviceID == nil || *repo.calledFilter.DeviceID != deviceID {
		t.Fatalf("expected DeviceID %s, got %v", deviceID, repo.calledFilter.DeviceID)
	}
	if len(repo.calledFilter.Severities) != 2 {
		t.Fatalf("expected 2 severities, got %d", len(repo.calledFilter.Severities))
	}
	if repo.calledFilter.From == nil || repo.calledFilter.To == nil {
		t.Fatalf("expected From and To to be set")
	}
	if repo.calledFilter.Page != 2 || repo.calledFilter.PageSize != 25 || repo.calledFilter.Offset != 25 {
		t.Fatalf("unexpected paging: page=%d size=%d offset=%d", repo.calledFilter.Page, repo.calledFilter.PageSize, repo.calledFilter.Offset)
	}
}

func TestListAlertsDeduplicatesSeverities(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	_, err := service.ListAlerts(context.Background(), uuid.New(), ListAlertsInput{
		Severities: []string{"warning", "warning", "critical", "warning"},
		Page:       1,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.calledFilter.Severities) != 2 {
		t.Fatalf("expected 2 unique severities, got %v", repo.calledFilter.Severities)
	}
}

func TestListAlertsTreatsEmptySeverityValueAsAbsent(t *testing.T) {
	repo := &queryRepoStub{}
	service := newQueryService(repo)

	_, err := service.ListAlerts(context.Background(), uuid.New(), ListAlertsInput{
		Severities: []string{""},
		Page:       1,
		PageSize:   20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.calledFilter.Severities != nil {
		t.Fatalf("expected nil severities, got %v", repo.calledFilter.Severities)
	}
}
