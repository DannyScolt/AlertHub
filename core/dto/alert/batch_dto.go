package alert

const BatchMaxSize = 100

type BatchRequest struct {
	Events []IngestRequest `json:"events" binding:"required"`
}

type BatchAcceptedAlert struct {
	Index   int    `json:"index" example:"0"`
	AlertID string `json:"alert_id" example:"9f3d2e1a-1234-4321-abcd-1234567890ab"`
}

type BatchError struct {
	Index   int    `json:"index" example:"5"`
	Code    string `json:"code" example:"INVALID_SEVERITY"`
	Message string `json:"message" example:"severity must be one of info, warning, critical"`
}

type BatchResponse struct {
	Accepted int                  `json:"accepted" example:"98"`
	Rejected int                  `json:"rejected" example:"2"`
	Alerts   []BatchAcceptedAlert `json:"alerts"`
	Errors   []BatchError         `json:"errors"`
}

type BatchEnvelopeResponse struct {
	Status  bool          `json:"status" example:"true"`
	Message string        `json:"message" example:"Batch processed"`
	Data    BatchResponse `json:"data"`
}
