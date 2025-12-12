package app

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type App struct {
	Leaderboard *LeaderboardService
	Redis       *redis.Client
}

func NewAppFromConfig(cfg *struct {
	RedisAddr string
	HTTPAddr  string
	Leagues   []struct {
		ID    int
		Min   int
		Max   int
		Names map[string]string
	}
}) (*App, error) {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	srv := NewLeaderboardService(rdb, cfg.Leagues)
	return &App{
		Leaderboard: srv,
		Redis:       rdb,
	}, nil
}
