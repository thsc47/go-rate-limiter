package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter/strategies"
)

type RedisLimiterStrategyMock struct {
	mock.Mock
}

func (m *RedisLimiterStrategyMock) Check(ctx context.Context, r *strategies.RateLimiterRequest) (*strategies.RateLimiterResult, error) {
	args := m.Called(ctx, r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*strategies.RateLimiterResult), args.Error(1)
}
