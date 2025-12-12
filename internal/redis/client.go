package redisclient

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	RDB *redis.Client
}

func New(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	return &Client{RDB: rdb}, nil
}
