package alert

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"

	TypeAutoEscalated = "auto_escalated"
)

type Alert struct {
	ID         uuid.UUID
	DeviceID   uuid.UUID
	ClientID   uuid.UUID
	Type       string
	Severity   Severity
	Message    string
	Payload    map[string]interface{}
	OccurredAt time.Time
	ReceivedAt time.Time
	CreatedAt  time.Time
}

func ValidSeverity(s Severity) bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func AllSeverities() []Severity {
	return []Severity{SeverityInfo, SeverityWarning, SeverityCritical}
}

func ValidateType(t string) bool {
	return strings.TrimSpace(t) != "" && len(t) <= 100
}

func ValidateMessage(m string) bool {
	return strings.TrimSpace(m) != ""
}
