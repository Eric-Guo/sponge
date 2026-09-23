package routers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Eric-Guo/sponge/pkg/errcode"
	"github.com/Eric-Guo/sponge/pkg/gin/handlerfunc"
	"github.com/Eric-Guo/sponge/pkg/gin/middleware"
	"github.com/Eric-Guo/sponge/pkg/gin/middleware/metrics"
	"github.com/Eric-Guo/sponge/pkg/gin/prof"
	"github.com/Eric-Guo/sponge/pkg/gin/swagger"
	"github.com/Eric-Guo/sponge/pkg/logger"

	"github.com/Eric-Guo/sponge/docs"
	"github.com/Eric-Guo/sponge/internal/config"
)

type routeFns = []func(r *gin.Engine, groupPathMiddlewares map[string][]gin.HandlerFunc, singlePathMiddlewares map[string][]gin.HandlerFunc)

var (
	// all route functions
	allRouteFns = make(routeFns, 0)
	// all middleware functions
	allMiddlewareFns = []func(c *middlewareConfig){}
)

// NewRouter_pbExample create a new router
func NewRouter_pbExample() *gin.Engine { //nolint
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Cors())

	if config.Get().HTTP.Timeout > 0 {
		// if you need more fine-grained control over your routes, set the timeout in your routes, unsetting the timeout globally here.
		r.Use(middleware.Timeout(time.Second * time.Duration(config.Get().HTTP.Timeout)))
	}

	// request id middleware
	r.Use(middleware.RequestID())

	// logger middleware, to print simple messages, replace middleware.Logging with middleware.SimpleLog
	r.Use(middleware.Logging(
		middleware.WithLog(logger.Get()),
		middleware.WithRequestIDFromContext(),
		middleware.WithIgnoreRoutes("/metrics"), // ignore path
	))

	// metrics middleware
	if config.Get().App.EnableMetrics {
		r.Use(metrics.Metrics(r,
			//metrics.WithMetricsPath("/metrics"),                // default is /metrics
			metrics.WithIgnoreStatusCodes(http.StatusNotFound), // ignore 404 status codes
		))
	}

	// limit middleware
	if config.Get().App.EnableLimit {
		r.Use(middleware.RateLimit(
		//middleware.WithWindow(time.Second*5), // default 10s
		//middleware.WithBucket(200), // default 100
		//middleware.WithCPUThreshold(900), // default 800
		))
	}

	// circuit breaker middleware
	if config.Get().App.EnableCircuitBreaker {
		r.Use(middleware.CircuitBreaker(
			//middleware.WithBreakerOption(
			//circuitbreaker.WithSuccess(75),           // default 60
			//circuitbreaker.WithRequest(100),          // default 100
			//circuitbreaker.WithBucket(20),            // default 10
			//circuitbreaker.WithWindow(time.Second*3), // default 3s
			//),
			//middleware.WithDegradeHandler(handler),              // Add degradation processing
			middleware.WithValidCode( // Add error codes to trigger circuit breaking
				errcode.InternalServerError.Code(),
				errcode.ServiceUnavailable.Code(),
			),
		))
	}

	// trace middleware
	if config.Get().App.EnableTrace {
		r.Use(middleware.Tracing(config.Get().App.Name))
	}

	// profile performance analysis
	if config.Get().App.EnableHTTPProfile {
		prof.Register(r, prof.WithIOWaitTime())
	}

	r.GET("/health", handlerfunc.CheckHealth)
	r.GET("/ping", handlerfunc.Ping)
	r.GET("/codes", handlerfunc.ListCodes)

	if config.Get().App.Env != "prod" {
		r.GET("/config", gin.WrapF(errcode.ShowConfig([]byte(config.Show()))))
		// access path /apis/swagger/index.html
		swagger.CustomRouter(r, "apis", docs.ApiDocs)
	}

	c := newMiddlewareConfig()

	// set up all middlewares
	for _, fn := range allMiddlewareFns {
		fn(c)
	}

	// register all routes
	for _, fn := range allRouteFns {
		fn(r, c.groupPathMiddlewares, c.singlePathMiddlewares)
	}

	return r
}

type middlewareConfig struct {
	groupPathMiddlewares  map[string][]gin.HandlerFunc // middleware functions corresponding to route group
	singlePathMiddlewares map[string][]gin.HandlerFunc // middleware functions corresponding to a single route
}

func newMiddlewareConfig() *middlewareConfig {
	return &middlewareConfig{
		groupPathMiddlewares:  make(map[string][]gin.HandlerFunc),
		singlePathMiddlewares: make(map[string][]gin.HandlerFunc),
	}
}

func (c *middlewareConfig) setGroupPath(groupPath string, handlers ...gin.HandlerFunc) { //nolint
	if groupPath == "" {
		return
	}
	if groupPath[0] != '/' {
		groupPath = "/" + groupPath
	}

	handlerFns, ok := c.groupPathMiddlewares[groupPath]
	if !ok {
		c.groupPathMiddlewares[groupPath] = handlers
		return
	}

	c.groupPathMiddlewares[groupPath] = append(handlerFns, handlers...)
}

func (c *middlewareConfig) setSinglePath(method string, singlePath string, handlers ...gin.HandlerFunc) { //nolint
	if method == "" || singlePath == "" {
		return
	}

	key := getSinglePathKey(method, singlePath)
	handlerFns, ok := c.singlePathMiddlewares[key]
	if !ok {
		c.singlePathMiddlewares[key] = handlers
		return
	}

	c.singlePathMiddlewares[key] = append(handlerFns, handlers...)
}

func getSinglePathKey(method string, singlePath string) string { //nolint
	return strings.ToUpper(method) + "->" + singlePath
}
