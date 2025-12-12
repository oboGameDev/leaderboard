package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	cfgpkg "github.com/oboGameDev/leaderboard/internal/config"
	"github.com/redis/go-redis/v9"
)

type LeaderboardService struct {
	rdb     *redis.Client
	leagues []League
}

func NewLeaderboardService(rdb *redis.Client, yamlLeagues []cfgpkg.LeagueYAML) *LeaderboardService {
	var leagues []League
	for _, l := range yamlLeagues {
		leagues = append(leagues, League{
			ID:    l.ID,
			Min:   l.Min,
			Max:   l.Max,
			Names: l.Names,
		})
	}
	return &LeaderboardService{
		rdb:     rdb,
		leagues: leagues,
	}
}

// helper keys
func leagueKey(id int) string            { return fmt.Sprintf("league:%d:lb", id) }
func userPointsKey(userID string) string { return fmt.Sprintf("user:%s:points", userID) }
func userLeagueKey(userID string) string { return fmt.Sprintf("user:%s:league", userID) }

// Determine league by points using configured leagues
func (s *LeaderboardService) determineLeague(points int) int {
	for _, l := range s.leagues {
		if l.Max == -1 {
			if points >= l.Min {
				return l.ID
			}
		} else if points >= l.Min && points <= l.Max {
			return l.ID
		}
	}
	if len(s.leagues) > 0 {
		return s.leagues[len(s.leagues)-1].ID
	}
	return 0
}

// UpdateUserPoints updates points (delta can be negative) and moves user between leaderboards
func (s *LeaderboardService) UpdateUserPoints(ctx context.Context, userID string, delta int64) (int64, int, error) {
	// read current
	curr, err := s.rdb.Get(ctx, userPointsKey(userID)).Int64()
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}
	newPoints := curr + delta
	if newPoints < 0 {
		newPoints = 0
	}
	newLeague := s.determineLeague(int(newPoints))
	oldLeague, _ := s.rdb.Get(ctx, userLeagueKey(userID)).Int()
	// atomic-ish via pipeline
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, userPointsKey(userID), newPoints, 0)
	if oldLeague != 0 && oldLeague != newLeague {
		pipe.ZRem(ctx, leagueKey(oldLeague), userID)
	}
	pipe.ZAdd(ctx, leagueKey(newLeague), redis.Z{
		Score:  float64(newPoints),
		Member: userID,
	})
	pipe.Set(ctx, userLeagueKey(userID), newLeague, 0)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, 0, err
	}
	return newPoints, newLeague, nil
}

func (s *LeaderboardService) AddUserPoints(ctx context.Context, userID string, delta int64) (int64, int, error) {
	if delta < 0 {
		return 0, 0, errors.New("delta must be positive")
	}
	return s.UpdateUserPoints(ctx, userID, delta)
}
func (s *LeaderboardService) RemoveUserPoints(ctx context.Context, userID string, delta int64) (int64, int, error) {
	if delta < 0 {
		return 0, 0, errors.New("delta must be positive")
	}
	return s.UpdateUserPoints(ctx, userID, -delta)
}

func (s *LeaderboardService) GetUserRank(ctx context.Context, leagueID int, userID string) (int64, error) {
	rank, err := s.rdb.ZRevRank(ctx, leagueKey(leagueID), userID).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, errors.New("user not found in leaderboard")
		}
		return 0, err
	}
	return rank + 1, nil
}

// Cursor helpers
func parseCursor(c string) (float64, string, error) {
	if c == "" {
		return 0, "", nil
	}
	parts := strings.SplitN(c, ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid cursor format")
	}
	score, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, "", err
	}
	return score, parts[1], nil
}
func buildCursor(score float64, userID string) string {
	if score == float64(int64(score)) {
		return fmt.Sprintf("%.0f:%s", score, userID)
	}
	return fmt.Sprintf("%f:%s", score, userID)
}

// GetLeagueLeaderboard returns items and next cursor (empty if no more)
func (s *LeaderboardService) GetLeagueLeaderboard(ctx context.Context, leagueID int, cursor string, limit int) ([]LeaderboardItem, string, error) {
	if limit <= 0 {
		return nil, "", errors.New("limit must be > 0")
	}
	key := leagueKey(leagueID)
	var zs []redis.Z

	if cursor == "" {
		res, err := s.rdb.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
		if err != nil {
			return nil, "", err
		}
		zs = res
	} else {
		lastScore, lastUser, err := parseCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		zr := &redis.ZRangeBy{
			Max:   fmt.Sprintf("%f", lastScore),
			Min:   "-inf",
			Count: int64(limit + 1),
		}
		res, err := s.rdb.ZRevRangeByScoreWithScores(ctx, key, zr).Result()
		if err != nil {
			return nil, "", err
		}
		start := 0
		if len(res) > 0 {
			if fmt.Sprint(res[0].Member) == lastUser && res[0].Score == lastScore {
				start = 1
			}
		}
		for i := start; i < len(res) && len(zs) < limit; i++ {
			zs = append(zs, res[i])
		}
	}

	items := make([]LeaderboardItem, 0, len(zs))
	for _, z := range zs {
		uid := fmt.Sprint(z.Member)
		rank, err := s.rdb.ZRevRank(ctx, key, uid).Result()
		if err != nil {
			continue
		}
		items = append(items, LeaderboardItem{
			UserID: uid,
			Points: z.Score,
			Rank:   rank + 1,
		})
	}

	next := ""
	if len(zs) > 0 {
		last := zs[len(zs)-1]
		next = buildCursor(last.Score, fmt.Sprint(last.Member))
	}
	return items, next, nil
}
