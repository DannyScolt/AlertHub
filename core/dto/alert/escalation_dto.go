package alert

import "time"

// EscalationPayload describes the source burst that produced an auto-escalated alert.
type EscalationPayload struct {
	SourceAlertIDs []string  `json:"source_alert_ids"`
	Count          int       `json:"count"`
	WindowSeconds  int       `json:"window_seconds"`
	Threshold      int       `json:"threshold"`
	DetectedAt     time.Time `json:"detected_at"`
}
