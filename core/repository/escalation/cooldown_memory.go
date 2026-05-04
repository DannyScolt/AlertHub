package escalation

import (
	"context"
	"sync"
	"time"
)

type memoryCooldownStore struct {
	mu      sync.Mutex
	expires map[CooldownKey]time.Time
	now     func() time.Time
}

func NewMemoryCooldownStore(now func() time.Time) CooldownStore {
	return &memoryCooldownStore{expires: make(map[CooldownKey]time.Time), now: now}
}

func (s *memoryCooldownStore) ClaimEscalation(_ context.Context, key CooldownKey, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if expiresAt, ok := s.expires[key]; ok && now.Before(expiresAt) {
		return false, nil
	}
	s.expires[key] = now.Add(ttl)
	return true, nil
}
