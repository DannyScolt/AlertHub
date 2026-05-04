package redis

import (
	"context"
	"testing"

	"alerthub/core/config"

	miniredis "github.com/alicebob/miniredis/v2"
)

func TestNewClientParsesURLWithPasswordAndPings(t *testing.T) {
	server := miniredis.RunT(t)
	server.RequireAuth("test-pass")

	client, err := NewClient(context.Background(), &config.Config{RedisURL: "redis://:test-pass@" + server.Addr() + "/0"})
	if err != nil {
		t.Fatalf("expected redis client, got %v", err)
	}
	defer client.Close()
}

func TestNewClientReturnsWrappedPingError(t *testing.T) {
	server := miniredis.RunT(t)
	addr := server.Addr()
	server.Close()

	_, err := NewClient(context.Background(), &config.Config{RedisURL: "redis://:test-pass@" + addr + "/0"})
	if err == nil {
		t.Fatal("expected ping error")
	}
}
