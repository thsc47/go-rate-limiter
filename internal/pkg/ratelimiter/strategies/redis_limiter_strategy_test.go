package strategies_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"

	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/mocks"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter/strategies"
)

type RedisLimiterStrategyTestSuite struct {
	suite.Suite
	Ctx             context.Context
	RedisClient     *redis.Client
	RedisClientMock redismock.ClientMock
	LoggerMock      *mocks.LoggerMock

	Strategy *strategies.RedisLimiterStrategy
}

func TestRedisLimiterStrategy(t *testing.T) {
	suite.Run(t, new(RedisLimiterStrategyTestSuite))
}

func (s *RedisLimiterStrategyTestSuite) SetupTest() {
	client, clientMock := redismock.NewClientMock()
	loggerMock := new(mocks.LoggerMock)

	loggerMock.On("GetLogger").Return(zerolog.Nop())

	s.Ctx = context.Background()
	s.LoggerMock = loggerMock
	s.RedisClient = client
	s.RedisClientMock = clientMock

	s.Strategy = strategies.NewRedisLimiterStrategy(client, zerolog.Nop(), timeNowMock)
}

func (s *RedisLimiterStrategyTestSuite) clearMocks() {
	s.RedisClientMock.ClearExpect()
	s.LoggerMock.ExpectedCalls = nil
}

func (s *RedisLimiterStrategyTestSuite) TestRedisLimiterStrategy() {
	key := "any-key"
	limit := int64(10)
	durationMillis := int64(1000)

	s.Run("Should allow request when key does not exists", func() {
		defer s.clearMocks()

		expectedTTL := time.Duration(durationMillis) * time.Millisecond

		s.RedisClientMock.ExpectGet(key).RedisNil()
		s.RedisClientMock.ExpectIncr(key).SetVal(1)
		s.RedisClientMock.ExpectTTL(key).SetVal(time.Duration(-1))
		s.RedisClientMock.ExpectExpire(key, time.Duration(durationMillis)).RedisNil()

		req := &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		}

		result, err := s.Strategy.Check(s.Ctx, req)

		s.NoError(err)
		s.NotNil(result)
		s.Equal(strategies.Allow, result.Result)
		s.Equal(limit, result.Limit)
		s.Equal(int64(1), result.Total)
		s.Equal(limit-1, result.Remaining)
		s.WithinDuration(timeNowMock().Add(expectedTTL), result.ExpiresAt, time.Second)
	})

	s.Run("Should allow request when key exists and count is less than limit", func() {
		defer s.clearMocks()

		expectedTTL := time.Duration(durationMillis) * time.Millisecond

		s.RedisClientMock.ExpectGet(key).SetVal("1")
		s.RedisClientMock.ExpectTTL(key).SetVal(expectedTTL)
		s.RedisClientMock.ExpectIncr(key).SetVal(2)

		req := &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		}

		result, err := s.Strategy.Check(s.Ctx, req)

		s.NoError(err)
		s.NotNil(result)
		s.Equal(strategies.Allow, result.Result)
		s.Equal(limit, result.Limit)
		s.Equal(int64(2), result.Total)
		s.Equal(limit-2, result.Remaining)
		s.WithinDuration(timeNowMock().Add(expectedTTL), result.ExpiresAt, time.Second)
	})

	s.Run("Should deny request when count is equal to limit", func() {
		defer s.clearMocks()

		expectedTTL := time.Duration(durationMillis) * time.Millisecond

		s.RedisClientMock.ExpectGet(key).SetVal("10")
		s.RedisClientMock.ExpectTTL(key).SetVal(expectedTTL)

		req := &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		}

		result, err := s.Strategy.Check(s.Ctx, req)

		s.NoError(err)
		s.NotNil(result)
		s.Equal(strategies.Deny, result.Result)
		s.Equal(limit, result.Limit)
		s.Equal(int64(10), result.Total)
		s.Equal(int64(0), result.Remaining)
		s.WithinDuration(timeNowMock().Add(expectedTTL), result.ExpiresAt, time.Second)
	})

	s.Run("Should deny request when incremented value is greater than limit", func() {
		defer s.clearMocks()

		expectedTTL := time.Duration(durationMillis) * time.Millisecond

		s.RedisClientMock.ExpectGet(key).SetVal("9")
		s.RedisClientMock.ExpectTTL(key).SetVal(expectedTTL)
		s.RedisClientMock.ExpectIncr(key).SetVal(11)

		req := &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		}

		result, err := s.Strategy.Check(s.Ctx, req)

		s.NoError(err)
		s.NotNil(result)
		s.Equal(strategies.Deny, result.Result)
		s.Equal(limit, result.Limit)
		s.Equal(int64(11), result.Total)
		s.Equal(int64(0), result.Remaining)
		s.WithinDuration(timeNowMock().Add(expectedTTL), result.ExpiresAt, time.Second)
	})

	s.Run("Should return error when pipeline exec fails", func() {
		defer s.clearMocks()

		s.RedisClientMock.ExpectGet(key).SetVal("1")
		s.RedisClientMock.ExpectTTL(key).SetErr(errors.New("any-error"))

		_, err := s.Strategy.Check(s.Ctx, &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		})

		s.Error(err)
		s.EqualError(err, "any-error")
	})

	s.Run("Should return error when get result fails", func() {
		defer s.clearMocks()

		s.RedisClientMock.ExpectGet(key).SetErr(errors.New("any-error"))
		s.RedisClientMock.ExpectTTL(key).SetVal(time.Duration(-1))

		_, err := s.Strategy.Check(s.Ctx, &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		})

		s.Error(err)
		s.EqualError(err, "any-error")
	})

	s.Run("Should return error when expire fails", func() {
		defer s.clearMocks()

		s.RedisClientMock.ExpectGet(key).SetVal("1")
		s.RedisClientMock.ExpectTTL(key).SetVal(time.Duration(-1))
		s.RedisClientMock.ExpectExpire(key, time.Duration(durationMillis)).SetErr(errors.New("any-error"))

		_, err := s.Strategy.Check(s.Ctx, &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		})

		s.Error(err)
		s.EqualError(err, "any-error")
	})

	s.Run("Should return error when increment fails", func() {
		defer s.clearMocks()

		expectedTTL := time.Duration(durationMillis) * time.Millisecond

		s.RedisClientMock.ExpectGet(key).SetVal("1")
		s.RedisClientMock.ExpectTTL(key).SetVal(expectedTTL)
		s.RedisClientMock.ExpectIncr(key).SetErr(errors.New("any-error"))

		_, err := s.Strategy.Check(s.Ctx, &strategies.RateLimiterRequest{
			Key:      key,
			Limit:    limit,
			Duration: time.Duration(durationMillis) * time.Millisecond,
		})

		s.Error(err)
		s.EqualError(err, "any-error")
	})
}

func timeNowMock() time.Time {
	return time.Date(2024, 2, 6, 0, 50, 0, 0, time.Local)
}
