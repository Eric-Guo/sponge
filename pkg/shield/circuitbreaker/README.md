## Circuit Breaker

### Principle

The Circuit Breaker is a **service protection mechanism** designed to prevent the continuous consumption of system resources and avoid cascading failures when calls to downstream services fail.

The core idea is similar to an **electrical fuse**:

* When downstream services fail frequently, the circuit breaker "trips the circuit" (rejects requests) to prevent system resources from being exhausted.
* After a period of time, the circuit breaker enters a **half-open state**, allowing a few test requests to pass through. If the service recovers, it "closes the circuit" and resumes normal operation.

This approach:
- Prevents cascading failures
- Fails fast to protect upstream resources
- Enables automatic recovery, improving system availability

<br>

### State Machine Mechanism

The circuit breaker primarily has **three states**:

```text
   ┌──────────┐         Error rate too high         ┌──────────┐
   │          │ ──────────────────────────────────▶ │          │
   │  Closed  │                                      │   Open   │
   │ (Closed) │ ◀────────────────────────────────── │ (Open)   │
   └──────────┘         Service recovered            └──────────┘
          │                                           ▲
          │                                           │
          │          Probe requests successful        │
          ▼                                           │
   ┌───────────────┐                                  │
   │   Half-Open   │ ─────────────────────────────────┘
   │ (Half-Open)   │
   └───────────────┘
```

* **Closed**: Normal state, allows all requests, tracks success rate
* **Open**: Rejects most requests, only occasionally allows test requests
* **Half-Open**: Allows a small number of probe requests; returns to Closed state if success rate recovers

<br>

### Implementation Mechanism

This implementation uses the **SRE adaptive circuit breaking algorithm** with the following mechanisms:

1. **Sliding Window Statistics (Rolling Window)**

   * Tracks request metrics over the most recent `window` period
   * Uses `bucket` partitioning to smooth data and avoid transient fluctuations

2. **Adaptive Checking**

   * Uses the formula `requests < k * accepts` to determine if circuit breaking should trigger
   * `k = 1 / success`, where `success` is the expected success rate threshold
   * Enters circuit breaking state when request volume is high and success rate is low

3. **Probabilistic Dropping**

   * Instead of outright rejection, randomly drops requests based on drop probability `dr`
   * Ensures some probe traffic continues, avoiding permanent circuit breaking

<br>

### Advantages

* **High Performance**: Low overhead based on sliding windows and atomic operations
* **Adaptive**: Automatically adjusts drop probability based on success rate and request volume
* **Smooth Circuit Breaking**: Uses probabilistic dropping instead of hard switches to avoid sudden traffic stops
* **Highly Configurable**: Supports adjustment of success rate threshold, minimum requests, window size, and bucket count

<br>

### Configuration Parameters

| Parameter  | Description                  | Default |
| ---------- | ---------------------------- | ------- |
| `success`  | Success rate threshold (0.6 = 60%) | `0.6`   |
| `request`  | Minimum requests to trigger circuit breaking | `100`   |
| `window`   | Statistical window size      | `3s`    |
| `bucket`   | Number of buckets in window (sliding bucket statistics) | `10`    |

<br>

### Circuit Breaker State Transition Conditions

1. **Closed → Open**

   * Request count exceeds `request`
   * Success rate falls below `success`

2. **Open → Half-Open**

   * `window` time period elapses
   * Allows a small number of probe requests

3. **Half-Open → Closed**

   * Probe requests succeed, service recovers

With this circuit breaker implementation, you can effectively protect services in high-concurrency distributed systems, avoid cascading failures, and improve overall system stability.

<br>

### Example of use

#### Gin circuit breaker middleware

```go
import "github.com/Eric-Guo/sponge/pkg/shield/circuitbreaker"

// CircuitBreaker a circuit breaker middleware
func CircuitBreaker(opts ...CircuitBreakerOption) gin.HandlerFunc {
	o := defaultCircuitBreakerOptions()
	o.apply(opts...)

	return func(c *gin.Context) {
		breaker := o.group.Get(c.FullPath()).(circuitbreaker.CircuitBreaker)
		if err := breaker.Allow(); err != nil {
			// NOTE: when client reject request locally, keep adding counter let the drop ratio higher.
			breaker.MarkFailed()
			response.Output(c, http.StatusServiceUnavailable, err.Error())
			c.Abort()
			return
		}

		c.Next()

		code := c.Writer.Status()
		// NOTE: need to check internal and service unavailable error, e.g. http.StatusInternalServerError
		_, isHit := o.validCodes[code]
		if isHit {
			breaker.MarkFailed()
		} else {
			breaker.MarkSuccess()
		}
	}
}
```

<br>

#### gRPC server circuit breaker interceptor

```go
import "github.com/Eric-Guo/sponge/pkg/shield/circuitbreaker"

// UnaryServerCircuitBreaker server-side unary circuit breaker interceptor
func UnaryServerCircuitBreaker(opts ...CircuitBreakerOption) grpc.UnaryServerInterceptor {
	o := defaultCircuitBreakerOptions()
	o.apply(opts...)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		breaker := o.group.Get(info.FullMethod).(circuitbreaker.CircuitBreaker)
		if err := breaker.Allow(); err != nil {
			// NOTE: when client reject request locally, keep adding let the drop ratio higher.
			breaker.MarkFailed()
			return nil, errcode.StatusServiceUnavailable.ToRPCErr(err.Error())
		}

		reply, err := handler(ctx, req)
		if err != nil {
			// NOTE: need to check internal and service unavailable error
			s, ok := status.FromError(err)
			_, isHit := o.validCodes[s.Code()]
			if ok && isHit {
				breaker.MarkFailed()
			} else {
				breaker.MarkSuccess()
			}
		}

		return reply, err
	}
}
```
