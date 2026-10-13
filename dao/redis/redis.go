package redis

import (
	"context"
	"fmt"
	"riff/conf"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
)

func InitRedis(cfg *conf.RedisConfig) (err error) {
	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	return nil
}

func Close() {
	if client != nil {
		_ = client.Close()
	}
}
