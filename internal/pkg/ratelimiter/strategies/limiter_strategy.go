package strategies

import (
	"context"
	"time"
)

// RateLimitResult represents the possible outcomes of a rate limit check
type RateLimitResult int

const (
	// Allow indicates the request should be allowed
	Allow RateLimitResult = 1
	// Deny indicates the request should be denied
	Deny RateLimitResult = -1
)

// RateLimitRequest encapsulates the parameters for a rate limit check
type RateLimitRequest struct {
	// Key is the unique identifier for rate limiting (e.g., IP address or API token)
	Key string
	// Limit is the maximum number of requests allowed within the Duration
	Limit int64
	// Duration is the time window for rate limiting
	Duration time.Duration
}

// RateLimitResponse contains the result of a rate limit check
type RateLimitResponse struct {
	// Result indicates whether the request is allowed or denied
	Result RateLimitResult
	// Limit is the maximum number of requests allowed
	Limit int64
	// Total is the current count of requests made
	Total int64
	// Remaining is the number of requests remaining in the current window
	Remaining int64
	// ExpiresAt is when the current rate limit window expires
	ExpiresAt time.Time
}

// RateLimiterStrategy defines the interface for rate limiting implementations
type RateLimiterStrategy interface {
	// Check evaluates whether a request should be allowed based on rate limiting rules
	// Returns RateLimitResponse with the result and current limits, or an error if the check fails
	Check(ctx context.Context, req *RateLimitRequest) (*RateLimitResponse, error)
}
