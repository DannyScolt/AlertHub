package common

type APIResponse struct {
	Status  bool        `json:"status" example:"true"`
	Message string      `json:"message" example:"Success"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorBody struct {
	Code    string      `json:"code" example:"VALIDATION_ERROR"`
	Details interface{} `json:"details,omitempty"`
}

type ErrorResponse struct {
	Status  bool      `json:"status" example:"false"`
	Message string    `json:"message" example:"Request validation failed"`
	Error   ErrorBody `json:"error"`
}

type FieldError struct {
	Field   string `json:"field" example:"email"`
	Message string `json:"message" example:"email is required"`
}

type PaginationMeta struct {
	Page        int   `json:"page" example:"1"`
	PageSize    int   `json:"page_size" example:"20"`
	Total       int64 `json:"total" example:"100"`
	TotalPages  int   `json:"total_pages" example:"5"`
	HasNext     bool  `json:"has_next" example:"true"`
	HasPrevious bool  `json:"has_previous" example:"false"`
}

type PaginatedResponse struct {
	Status     bool           `json:"status" example:"true"`
	Message    string         `json:"message" example:"Success"`
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}
