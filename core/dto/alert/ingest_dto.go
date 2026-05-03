package alert

import "time"

type IngestRequest struct {
	Type       string                 `json:"type" binding:"required" example:"high_temperature"`
	Severity   string                 `json:"severity" binding:"required" example:"warning"`
	Message    string                 `json:"message" binding:"required" example:"Temperature exceeded 80°C"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	OccurredAt *time.Time             `json:"occurred_at,omitempty" example:"2026-05-03T12:00:00Z"`
}

type IngestResponse struct {
	AlertID    string    `json:"alert_id" example:"9f3d2e1a-1234-4321-abcd-1234567890ab"`
	ReceivedAt time.Time `json:"received_at" example:"2026-05-03T12:00:00.123Z"`
}

type IngestEnvelopeResponse struct {
	Status  bool           `json:"status" example:"true"`
	Message string         `json:"message" example:"Event accepted"`
	Data    IngestResponse `json:"data"`
}
