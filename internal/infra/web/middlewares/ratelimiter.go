package middlewares

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/logger"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter"
	limiter "github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter/strategies"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/responsehandler"
)

const (
	// RateLimitHeaders
	headerRateLimitLimit     = "X-RateLimit-Limit"
	headerRateLimitRemaining = "X-RateLimit-Remaining"
	headerRateLimitReset     = "X-RateLimit-Reset"
)

type RateLimiterMiddlewareInterface interface {
	Handle(next http.Handler) http.Handler
}

// RateLimiterMiddleware handles rate limiting for HTTP requests
type RateLimiterMiddleware struct {
	logger          zerolog.Logger
	responseHandler responsehandler.WebResponseHandlerInterface
	limiter         ratelimiter.RateLimiterInterface
}

// NewRateLimiterMiddleware creates a new rate limiter middleware
func NewRateLimiterMiddleware(
	logger logger.LoggerInterface,
	responseHandler responsehandler.WebResponseHandlerInterface,
	limiter ratelimiter.RateLimiterInterface,
) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		logger:          logger.GetLogger(),
		responseHandler: responseHandler,
		limiter:         limiter,
	}
}

// Handle implements the middleware handler for rate limiting
func (rlm *RateLimiterMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, err := rlm.limiter.Check(r.Context(), r)
		if err != nil {
			rlm.logger.Error().Err(err).Msg("failed to check rate limit")
			rlm.responseHandler.RespondWithError(
				w,
				http.StatusInternalServerError,
				errors.Join(errors.New("failed to check rate limit"), err),
			)
			return
		}

		// Always set rate limit headers
		w.Header().Set(headerRateLimitLimit, strconv.FormatInt(result.Limit, 10))
		w.Header().Set(headerRateLimitRemaining, strconv.FormatInt(result.Remaining, 10))
		w.Header().Set(headerRateLimitReset, strconv.FormatInt(result.ExpiresAt.Unix(), 10))

		rlm.logger.Debug().
			Int64("limit", result.Limit).
			Int64("remaining", result.Remaining).
			Time("expires_at", result.ExpiresAt).
			Msg("rate limit check result")

		if result.Result == limiter.Deny {
			rlm.responseHandler.RespondWithError(
				w,
				http.StatusTooManyRequests,
				limiter.ErrRateLimitExceeded,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
