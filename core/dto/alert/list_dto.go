package alert

import "time"

type AlertResponse struct {
	ID         string                 `json:"id" example:"9f3d2e1a-1234-4321-abcd-1234567890ab"`
	DeviceID   string                 `json:"device_id" example:"4d285f4b-2a87-4a86-a5b8-05b09c6d1234"`
	Type       string                 `json:"type" example:"high_temperature"`
	Severity   string                 `json:"severity" example:"warning"`
	Message    string                 `json:"message" example:"Temperature exceeded 80°C"`
	Payload    map[string]interface{} `json:"payload"`
	OccurredAt time.Time              `json:"occurred_at"`
	ReceivedAt time.Time              `json:"received_at"`
}

type AlertPaginationMeta struct {
	Page        int   `json:"page" example:"1"`
	PageSize    int   `json:"page_size" example:"20"`
	Total       int64 `json:"total" example:"100"`
	TotalPages  int   `json:"total_pages" example:"5"`
	HasNext     bool  `json:"has_next" example:"true"`
	HasPrevious bool  `json:"has_previous" example:"false"`
}

type ListAlertsResponse struct {
	Status     bool                `json:"status" example:"true"`
	Message    string              `json:"message" example:"Alerts retrieved successfully"`
	Data       []AlertResponse     `json:"data"`
	Pagination AlertPaginationMeta `json:"pagination"`
}
