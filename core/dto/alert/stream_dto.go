package alert

import "time"

type StreamConnectedEvent struct {
	ClientID  string    `json:"client_id" example:"4d285f4b-2a87-4a86-a5b8-05b09c6d1234"`
	Timestamp time.Time `json:"timestamp" example:"2026-05-03T12:00:00Z"`
}

type StreamAlertEvent struct {
	ID         string                 `json:"id" example:"9f3d2e1a-1234-4321-abcd-1234567890ab"`
	DeviceID   string                 `json:"device_id" example:"4d285f4b-2a87-4a86-a5b8-05b09c6d1234"`
	Type       string                 `json:"type" example:"high_temperature"`
	Severity   string                 `json:"severity" example:"warning"`
	Message    string                 `json:"message" example:"Temperature exceeded 80°C"`
	Payload    map[string]interface{} `json:"payload"`
	OccurredAt time.Time              `json:"occurred_at" example:"2026-05-03T12:00:00Z"`
	ReceivedAt time.Time              `json:"received_at" example:"2026-05-03T12:00:00.123Z"`
}

type StreamHeartbeatEvent struct {
	Timestamp time.Time `json:"timestamp" example:"2026-05-03T12:00:30Z"`
}
