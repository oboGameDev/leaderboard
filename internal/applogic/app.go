package app

import (
	"context"
	"fmt"
	"time"

	cfgpkg "github.com/oboGameDev/leaderboard/internal/config"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Leaderboard *LeaderboardService
	Redis       *redis.Client
}

func NewAppFromConfig(cfg *cfgpkg.Config) (*App, error) {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: 0})
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

func (a *App) GetLeagueLeaderboard(ctx context.Context, leagueID int, cursor string, limit int) ([]LeaderboardItem, string, error) {
	return a.Leaderboard.GetLeagueLeaderboard(ctx, leagueID, cursor, limit)
}

func (a *App) GetUserRank(ctx context.Context, leagueID int, userID string) (int64, error) {
	return a.Leaderboard.GetUserRank(ctx, leagueID, userID)
}
func (a *App) AddUserPoints(ctx context.Context, userID string, delta int64) (int64, int, error) {
	return a.Leaderboard.AddUserPoints(ctx, userID, delta)
}
