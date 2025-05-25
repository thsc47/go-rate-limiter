package strategies

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

var (
	_ RateLimiterStrategy = &RedisRateLimiter{}

	// ErrRateLimitExceeded indicates that the rate limit has been exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	// ErrInvalidTTL indicates an invalid or missing TTL for the rate limit key
	ErrInvalidTTL = errors.New("invalid TTL for rate limit key")
)

const (
	keyWithoutTTL = -1
	keyNotFound   = -2
)

// RedisRateLimiter implements rate limiting using Redis as a backend
type RedisRateLimiter struct {
	client *redis.Client
	logger zerolog.Logger
	clock  func() time.Time
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(
	client *redis.Client,
	logger zerolog.Logger,
	clock func() time.Time,
) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		logger: logger,
		clock:  clock,
	}
}

// Check implements the RateLimiterStrategy interface for Redis
func (rl *RedisRateLimiter) Check(ctx context.Context, req *RateLimitRequest) (*RateLimitResponse, error) {
	pipe := rl.client.Pipeline()
	getCmd := pipe.Get(ctx, req.Key)
	ttlCmd := pipe.TTL(ctx, req.Key)

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	ttlDuration := req.Duration
	ttl, err := ttlCmd.Result()
	if err == nil && ttl != keyWithoutTTL && ttl != keyNotFound {
		ttlDuration = ttl
	} else {
		if err := rl.client.Expire(ctx, req.Key, req.Duration).Err(); err != nil {
			rl.logger.Error().Err(err).Msg("failed to set key expiration")
			return nil, err
		}
	}

	currentCount, err := getCmd.Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	// If key doesn't exist or error occurred, assume count is 0
	if errors.Is(err, redis.Nil) {
		currentCount = 0
	}

	expiresAt := rl.clock().Add(ttlDuration)

	// Fast path: if current count already exceeds limit, return immediately
	if currentCount >= req.Limit {
		return &RateLimitResponse{
			Result:    Deny,
			Total:     currentCount,
			Limit:     req.Limit,
			Remaining: 0,
			ExpiresAt: expiresAt,
		}, nil
	}

	// Increment the counter atomically
	nextCount, err := rl.client.Incr(ctx, req.Key).Result()
	if err != nil {
		return nil, err
	}

	// Check if the increment pushed us over the limit
	if nextCount > req.Limit {
		return &RateLimitResponse{
			Result:    Deny,
			Total:     nextCount,
			Limit:     req.Limit,
			Remaining: 0,
			ExpiresAt: expiresAt,
		}, nil
	}

	return &RateLimitResponse{
		Result:    Allow,
		Total:     nextCount,
		Limit:     req.Limit,
		Remaining: req.Limit - nextCount,
		ExpiresAt: expiresAt,
	}, nil
}
