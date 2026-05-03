package client

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
