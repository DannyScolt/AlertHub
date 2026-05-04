package redis

import (
	"context"
	"fmt"
	"log"

	"alerthub/core/config"

	redislib "github.com/redis/go-redis/v9"
)

func NewClient(ctx context.Context, cfg *config.Config) (*redislib.Client, error) {
	options, err := redislib.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redislib.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	log.Printf("redis ping ok: addr=%s", options.Addr)
	return client, nil
}
