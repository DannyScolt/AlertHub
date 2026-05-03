package refresh_token

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID           uuid.UUID
	ClientID     uuid.UUID
	TokenHash    string
	TokenFamily  uuid.UUID
	ParentID     *uuid.UUID
	ReplacedByID *uuid.UUID
	ExpiresAt    time.Time
	CreatedAt    time.Time
	LastUsedAt   *time.Time
	RevokedAt    *time.Time
	RevokeReason *string
	UserAgent    *string
	IPAddress    *net.IP
}

func (t RefreshToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

func (t RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t RefreshToken) IsReplaced() bool {
	return t.ReplacedByID != nil
}
