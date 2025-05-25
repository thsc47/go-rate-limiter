package dependencyinjector

import (
	"time"

	"github.com/mathcale/goexpert-rate-limiter-challenge/config"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/infra/database"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/infra/web"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/infra/web/handlers"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/infra/web/middlewares"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/logger"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/ratelimiter/strategies"
	"github.com/mathcale/goexpert-rate-limiter-challenge/internal/pkg/responsehandler"
)

// DependencyInjector handles the creation and wiring of application dependencies
type DependencyInjector struct {
	config *config.Conf
}

// Dependencies contains all the application dependencies
type Dependencies struct {
	Logger                logger.LoggerInterface
	ResponseHandler       responsehandler.WebResponseHandlerInterface
	HelloWebHandler       handlers.HelloWebHandlerInterface
	RateLimiterMiddleware middlewares.RateLimiterMiddlewareInterface
	WebServer             web.WebServerInterface
	RedisDatabase         database.RedisDatabaseInterface
	RateLimiter           ratelimiter.RateLimiter
	RateLimiterStrategy   strategies.RateLimiterStrategy
}

// NewDependencyInjector creates a new dependency injector
func NewDependencyInjector(c *config.Conf) *DependencyInjector {
	return &DependencyInjector{
		config: c,
	}
}

// Inject creates and wires all application dependencies
func (di *DependencyInjector) Inject() (*Dependencies, error) {
	logger := logger.NewLogger(di.config.LogLevel)
	responseHandler := responsehandler.NewWebResponseHandler()

	redisDB, err := database.NewRedisDatabase(*di.config, logger.GetLogger())
	if err != nil {
		return nil, err
	}

	rateLimiterStrategy := strategies.NewRedisRateLimiter(
		redisDB.Client,
		logger.GetLogger(),
		time.Now,
	)

	rateLimiter := ratelimiter.NewHTTPRateLimiter(
		logger,
		rateLimiterStrategy,
		di.config.RateLimiterIPMaxRequests,
		di.config.RateLimiterTokenMaxRequests,
		di.config.RateLimiterTimeWindowMilliseconds,
	)

	helloWebHandler := handlers.NewHelloWebHandler(responseHandler)
	rateLimiterMiddleware := middlewares.NewRateLimiterMiddleware(logger, responseHandler, rateLimiter)

	webRouter := web.NewWebRouter(helloWebHandler, rateLimiterMiddleware)
	webServer := web.NewWebServer(
		di.config.WebServerPort,
		logger.GetLogger(),
		webRouter.Build(),
		webRouter.BuildMiddlewares(),
	)

	return &Dependencies{
		Logger:                logger,
		ResponseHandler:       responseHandler,
		HelloWebHandler:       helloWebHandler,
		RateLimiterMiddleware: rateLimiterMiddleware,
		WebServer:             webServer,
		RedisDatabase:         redisDB,
		RateLimiter:           rateLimiter,
		RateLimiterStrategy:   rateLimiterStrategy,
	}, nil
}
