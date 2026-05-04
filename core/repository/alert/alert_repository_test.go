package alert

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildAlertListQueriesAppliesSearchToListAndCount(t *testing.T) {
	search := "smoke"
	queries := buildAlertListQueries(ListFilter{Search: &search})

	assertSearchQuery(t, queries.List)
	assertSearchQuery(t, queries.Count)
	if !strings.Contains(queries.List, "ORDER BY alerts.occurred_at DESC, alerts.id DESC") {
		t.Fatalf("expected list query to preserve ordering, got %s", queries.List)
	}
}

func TestBuildAlertListQueriesUsesContiguousPaginationPlaceholdersForTextSearch(t *testing.T) {
	search := "smoke"
	queries := buildAlertListQueries(ListFilter{Search: &search})

	if !strings.Contains(queries.List, "LIMIT $7 OFFSET $8") {
		t.Fatalf("expected text search pagination placeholders to follow search arg, got %s", queries.List)
	}
}

func TestBuildAlertListQueriesMatchesExactDeviceIDWhenSearchIsUUID(t *testing.T) {
	search := uuid.New().String()
	queries := buildAlertListQueries(ListFilter{Search: &search})

	if !strings.Contains(queries.List, "OR alerts.device_id = $7") {
		t.Fatalf("expected list query to match exact device id, got %s", queries.List)
	}
	if !strings.Contains(queries.Count, "OR alerts.device_id = $7") {
		t.Fatalf("expected count query to match exact device id, got %s", queries.Count)
	}
}

func assertSearchQuery(t *testing.T, query string) {
	t.Helper()
	expected := []string{
		"FROM alerts JOIN devices ON devices.id = alerts.device_id AND devices.client_id = $1",
		"alerts.client_id = $1",
		"alerts.message ILIKE $6",
		"alerts.type ILIKE $6",
		"devices.name ILIKE $6",
	}
	for _, part := range expected {
		if !strings.Contains(query, part) {
			t.Fatalf("expected query to contain %q, got %s", part, query)
		}
	}
}
