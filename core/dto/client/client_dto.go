package client

import "time"

type ClientResponse struct {
	ID        string    `json:"id" example:"8c5c8a5e-6b74-48b4-9e0b-85a0c9ef1234"`
	Email     string    `json:"email" example:"client@example.com"`
	Name      string    `json:"name" example:"Acme Inc"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MeResponse struct {
	Status  bool           `json:"status" example:"true"`
	Message string         `json:"message" example:"Client retrieved successfully"`
	Data    ClientResponse `json:"data"`
}
