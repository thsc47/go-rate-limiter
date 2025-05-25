package ratelimiter

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	rip "github.com/vikram1565/request-ip"

	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/logger"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter/strategies"
)

const (
	// DefaultHeaderAPIKey is the default header name for API key authentication
	DefaultHeaderAPIKey = "API_KEY"
)

// RateLimiter defines the interface for rate limiting requests
type RateLimiter interface {
	// Check evaluates if a request should be allowed based on rate limiting rules
	Check(ctx context.Context, r *http.Request) (*strategies.RateLimitResponse, error)
}

// HTTPRateLimiter implements rate limiting for HTTP requests
type HTTPRateLimiter struct {
	logger              zerolog.Logger
	strategy            strategies.RateLimiterStrategy
	maxRequestsPerIP    int64
	maxRequestsPerToken int64
	timeWindow          time.Duration
}

// NewHTTPRateLimiter creates a new HTTP rate limiter
func NewHTTPRateLimiter(
	logger logger.LoggerInterface,
	strategy strategies.RateLimiterStrategy,
	ipMaxReqs int,
	tokenMaxReqs int,
	timeWindowMillis int,
) *HTTPRateLimiter {
	return &HTTPRateLimiter{
		logger:              logger.GetLogger(),
		strategy:            strategy,
		maxRequestsPerIP:    int64(ipMaxReqs),
		maxRequestsPerToken: int64(tokenMaxReqs),
		timeWindow:          time.Duration(timeWindowMillis) * time.Millisecond,
	}
}

// Check implements the RateLimiter interface for HTTP requests
func (rl *HTTPRateLimiter) Check(ctx context.Context, r *http.Request) (*strategies.RateLimitResponse, error) {
	var (
		key   string
		limit int64
	)

	// Determine rate limit key and limit based on presence of API key
	if apiKey := r.Header.Get(DefaultHeaderAPIKey); apiKey != "" {
		key = apiKey
		limit = rl.maxRequestsPerToken
	} else {
		key = rip.GetClientIP(r)
		limit = rl.maxRequestsPerIP
	}

	rl.logger.Debug().
		Str("key", key).
		Int64("limit", limit).
		Dur("window", rl.timeWindow).
		Msg("checking rate limit")

	req := &strategies.RateLimitRequest{
		Key:      key,
		Limit:    limit,
		Duration: rl.timeWindow,
	}

	result, err := rl.strategy.Check(ctx, req)
	if err != nil {
		rl.logger.Error().Err(err).
			Str("key", key).
			Int64("limit", limit).
			Msg("failed to check rate limit")
		return nil, err
	}

	return result, nil
}
