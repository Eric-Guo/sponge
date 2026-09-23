## Adaptive Rate Limiting

The mechanism of adaptive rate limiting is derived from Google BBR's congestion control idea, and its essence is:

1. **If the system is healthy, don't block traffic**

   * The purpose of rate limiting is not to "shave peaks" but to "protect the system."
   * If the CPU load is not high, even if there are many requests, it means the system can still handle them, and there is no need to reject requests.
   * Otherwise, it would waste resources and reduce throughput.

2. **Only perform rate limiting when the system is about to be overwhelmed**

   * Requests are only dropped when the CPU exceeds the threshold and the current number of concurrent requests is greater than the safe upper limit calculated from historical data.
   * At this point, the purpose of rate limiting is to prevent a cascade failure and pull the system back from the brink of "overload → collapse."

The advantages of adaptive rate limiting are:

* **Protects system stability**: Requests are only dropped when the CPU is under high load, ensuring maximum throughput.
* **High-concurrency API services**: It can accept requests to the greatest extent while ensuring performance.

> Note: If the stress test scenario involves I/O-bound requests (with very low CPU usage), the rate limiting will hardly trigger and may appear "ineffective."

<br>

### Example of use

#### Gin ratelimit middleware

```go
import (
	rl "github.com/Eric-Guo/sponge/pkg/shield/ratelimit"
)

func RateLimit(opts ...RateLimitOption) gin.HandlerFunc {
	o := defaultRatelimitOptions()
	o.apply(opts...)
	limiter := rl.NewLimiter(
		rl.WithWindow(o.window),
		rl.WithBucket(o.bucket),
		rl.WithCPUThreshold(o.cpuThreshold),
		rl.WithCPUQuota(o.cpuQuota),
	)

	return func(c *gin.Context) {
		done, err := limiter.Allow()
		if err != nil {
			response.Output(c, http.StatusTooManyRequests, err.Error())
			c.Abort()
			return
		}

		c.Next()

		done(rl.DoneInfo{Err: c.Request.Context().Err()})
	}
}
```

<br>

#### gRPC ratelimit interceptor

```go
import (
	rl "github.com/Eric-Guo/sponge/pkg/shield/ratelimit"
)


func UnaryServerRateLimit(opts ...RatelimitOption) grpc.UnaryServerInterceptor {
	o := defaultRatelimitOptions()
	o.apply(opts...)
	limiter := rl.NewLimiter(
		rl.WithWindow(o.window),
		rl.WithBucket(o.bucket),
		rl.WithCPUThreshold(o.cpuThreshold),
		rl.WithCPUQuota(o.cpuQuota),
	)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		done, err := limiter.Allow()
		if err != nil {
			return nil, errcode.StatusLimitExceed.ToRPCErr(err.Error())
		}

		reply, err := handler(ctx, req)
		done(rl.DoneInfo{Err: err})
		return reply, err
	}
}
```
