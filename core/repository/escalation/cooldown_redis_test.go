package escalation

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redislib "github.com/redis/go-redis/v9"
)

func TestRedisCooldownClaimsFirstTimeAgainstMiniredis(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("test-pass")
	client := redisClient(t, server)
	store := NewRedisCooldownStore(client)

	claimed, err := store.ClaimEscalation(context.Background(), cooldownKey(), time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to win")
	}
}

func TestRedisCooldownRejectsSecondClaimWithinTTL(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("test-pass")
	client := redisClient(t, server)
	store := NewRedisCooldownStore(client)
	key := cooldownKey()
	_, _ = store.ClaimEscalation(context.Background(), key, time.Minute)

	claimed, err := store.ClaimEscalation(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claimed {
		t.Fatal("expected second claim to lose")
	}
}

func TestRedisCooldownAllowsAfterTTLViaMiniredisFastForward(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("test-pass")
	client := redisClient(t, server)
	store := NewRedisCooldownStore(client)
	key := cooldownKey()
	_, _ = store.ClaimEscalation(context.Background(), key, time.Minute)
	server.FastForward(time.Minute + time.Second)

	claimed, err := store.ClaimEscalation(context.Background(), key, time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !claimed {
		t.Fatal("expected claim after ttl to win")
	}
}

func TestRedisCooldownReturnsErrorWhenServerDown(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("test-pass")
	client := redisClient(t, server)
	server.Close()
	store := NewRedisCooldownStore(client)

	_, err := store.ClaimEscalation(context.Background(), cooldownKey(), time.Minute)
	if err == nil {
		t.Fatal("expected redis error")
	}
}

func redisClient(t *testing.T, server *miniredis.Miniredis) *redislib.Client {
	t.Helper()
	options, err := redislib.ParseURL("redis://:test-pass@" + server.Addr() + "/0")
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	client := redislib.NewClient(options)
	t.Cleanup(func() { client.Close() })
	return client
}
