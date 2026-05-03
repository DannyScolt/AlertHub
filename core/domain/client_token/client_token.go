package client_token

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type ClientToken struct {
	ID           uuid.UUID
	ClientID     uuid.UUID
	Name         string
	TokenHash    string
	TokenFamily  uuid.UUID
	Abilities    []string
	ParentID     *uuid.UUID
	ReplacedByID *uuid.UUID
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastUsedAt   *time.Time
	RevokedAt    *time.Time
	RevokeReason *string
	UserAgent    *string
	IPAddress    *net.IP
}

func (t ClientToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

func (t ClientToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t ClientToken) IsReplaced() bool {
	return t.ReplacedByID != nil
}
