package escalation

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CooldownKey struct {
	DeviceID  uuid.UUID
	AlertType string
}

type CooldownStore interface {
	ClaimEscalation(ctx context.Context, key CooldownKey, ttl time.Duration) (bool, error)
}
