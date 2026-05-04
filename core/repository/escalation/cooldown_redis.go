package escalation

import (
	"context"
	"fmt"
	"time"

	redislib "github.com/redis/go-redis/v9"
)

type redisCooldownStore struct {
	client *redislib.Client
}

func NewRedisCooldownStore(client *redislib.Client) CooldownStore {
	return &redisCooldownStore{client: client}
}

func (s *redisCooldownStore) ClaimEscalation(ctx context.Context, key CooldownKey, ttl time.Duration) (bool, error) {
	claimed, err := s.client.SetNX(ctx, redisCooldownKey(key), "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("claim escalation cooldown: %w", err)
	}
	return claimed, nil
}

func redisCooldownKey(key CooldownKey) string {
	return fmt.Sprintf("escalation:cooldown:%s:%s", key.DeviceID, key.AlertType)
}
