package device

import "time"

type CreateDeviceRequest struct {
	Name     string                 `json:"name" binding:"required" example:"Warehouse Temperature Sensor"`
	Type     string                 `json:"type" binding:"required" example:"temperature_sensor"`
	Status   string                 `json:"status" example:"active"`
	Tags     []string               `json:"tags" example:"warehouse,floor-1"`
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateDeviceRequest struct {
	Name     *string                `json:"name,omitempty" example:"Warehouse Temperature Sensor v2"`
	Type     *string                `json:"type,omitempty" example:"temperature_sensor"`
	Status   *string                `json:"status,omitempty" example:"maintenance"`
	Tags     []string               `json:"tags,omitempty" example:"warehouse,floor-1"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DeviceResponse struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Status     string                 `json:"status"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
	LastSeenAt *time.Time             `json:"last_seen_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	DeletedAt  *time.Time             `json:"deleted_at,omitempty"`
}

type DeviceWithAPIKeyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeleteDeviceResponse struct {
	ID         string    `json:"id"`
	DeletedAt  time.Time `json:"deleted_at"`
	PurgeAfter time.Time `json:"purge_after"`
}

type RotateDeviceAPIKeyResponse struct {
	ID        string    `json:"id"`
	APIKey    string    `json:"api_key"`
	RotatedAt time.Time `json:"rotated_at"`
}

type DeviceEnvelopeResponse struct {
	Status  bool           `json:"status" example:"true"`
	Message string         `json:"message" example:"Device retrieved successfully"`
	Data    DeviceResponse `json:"data"`
}

type CreateDeviceEnvelopeResponse struct {
	Status  bool                     `json:"status" example:"true"`
	Message string                   `json:"message" example:"Device created successfully"`
	Data    DeviceWithAPIKeyResponse `json:"data"`
}

type PaginationMeta struct {
	Page        int   `json:"page" example:"1"`
	PageSize    int   `json:"page_size" example:"20"`
	Total       int64 `json:"total" example:"100"`
	TotalPages  int   `json:"total_pages" example:"5"`
	HasNext     bool  `json:"has_next" example:"true"`
	HasPrevious bool  `json:"has_previous" example:"false"`
}

type ListDevicesResponse struct {
	Status     bool             `json:"status" example:"true"`
	Message    string           `json:"message" example:"Devices retrieved successfully"`
	Data       []DeviceResponse `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
}

type DeleteDeviceEnvelopeResponse struct {
	Status  bool                 `json:"status" example:"true"`
	Message string               `json:"message" example:"Device deleted successfully"`
	Data    DeleteDeviceResponse `json:"data"`
}

type RotateDeviceAPIKeyEnvelopeResponse struct {
	Status  bool                       `json:"status" example:"true"`
	Message string                     `json:"message" example:"Device API key rotated successfully"`
	Data    RotateDeviceAPIKeyResponse `json:"data"`
}
