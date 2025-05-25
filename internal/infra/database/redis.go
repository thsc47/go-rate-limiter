package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/mathcale/goexpert-rate-limiter-challenge/config"
)

type RedisDatabaseInterface interface{}

type RedisDatabase struct {
	Client *redis.Client
}

func NewRedisDatabase(
	cfg config.Conf,
	logger zerolog.Logger,
) (*RedisDatabase, error) {
	addr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)

	logger.Debug().Msgf("Connecting to Redis on [%s]", addr)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}

	logger.Debug().Msgf("Redis successfully connected")

	return &RedisDatabase{
		Client: client,
	}, nil
}
