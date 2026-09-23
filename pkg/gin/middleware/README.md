## middleware

Common gin middleware libraries, including:

- [Logging](README.md#logging-middleware)
- [Cors](README.md#allow-cross-domain-requests-middleware)
- [Rate limiter](README.md#rate-limiter-middleware)
- [Circuit breaker](README.md#circuit-breaker-middleware)
- [JWT authorization](README.md#jwt-authorization-middleware)
- [Tracing](README.md#tracing-middleware)
- [Metrics](README.md#metrics-middleware)
- [Request id](README.md#request-id-middleware)
- [Timeout](README.md#timeout-middleware)
 
<br>

## Example of use

### Logging middleware

You can set the maximum length for printing, add a request id field, ignore print path, customize [zap](https://github.com/uber-go/zap) log.

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    // Print input parameters and return results
    // Case 1: default
    {
        r.Use(middleware.Logging()
    }
    // Case 2: custom
    {
        r.Use(middleware.Logging(
            middleware.WithLog(logger.Get()))
            middleware.WithMaxLen(400),
            middleware.WithRequestIDFromHeader(),
            //middleware.WithRequestIDFromContext(),
            //middleware.WithIgnoreRoutes("/hello"),
        ))
    }

    /*******************************************
    TIP: You can use middleware.SimpleLog instead of
           middleware.Logging, it only prints return results
    *******************************************/

    // ......
    return r
}
```

<br>

### Allow cross-domain requests middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    r.Use(middleware.Cors())

    // ......
    return r
}
```

<br>

### Rate limiter middleware

Adaptive flow limitation based on hardware resources.

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    // Case 1: default
    r.Use(middleware.RateLimit())

    // Case 2: custom
    r.Use(middleware.RateLimit(
        middleware.WithWindow(time.Second*10),
        middleware.WithBucket(1000),
        middleware.WithCPUThreshold(100),
        middleware.WithCPUQuota(0.5),
    ))

    // ......
    return r
}
```

<br>

### Circuit Breaker middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    r.Use(middleware.CircuitBreaker(
        //middleware.WithValidCode(http.StatusRequestTimeout), // add error code 408 for circuit breaker
        //middleware.WithDegradeHandler(handler), // add custom degrade handler
        //middleware.WithBreakerOption(
            //circuitbreaker.WithSuccess(75),           // default 60
            //circuitbreaker.WithRequest(200),          // default 100
            //circuitbreaker.WithBucket(20),            // default 10
            //circuitbreaker.WithWindow(time.Second*5), // default 3s
        //),
    ))

    // ......
    return r
}
```

<br>

### JWT authorization middleware

There are two usage examples available:

1. **Example One**: This example adopts a highly abstracted design, making it simpler and more convenient to use. Click to view the example at [pkg/gin/middleware/auth](https://github.com/Eric-Guo/sponge/tree/thruster_generate/pkg/gin/middleware/auth#example-of-use). Requires sponge version `v1.13.2+`.
2. **Example Two**: This example offers greater flexibility and is suitable for scenarios requiring custom implementations. The example code is as follows:

    ```go
    package main
    
    import (
        "time"
        "github.com/gin-gonic/gin"
        "github.com/Eric-Guo/sponge/pkg/gin/middleware"
        "github.com/Eric-Guo/sponge/pkg/gin/response"
        "github.com/Eric-Guo/sponge/pkg/jwt"
    )
    
    func main() {
        r := gin.Default()
    
        g := r.Group("/api/v1")
    
        // Case 1: default jwt options, signKey, signMethod(HS256), expiry time(24 hour)
        {
            r.POST("/auth/login", LoginDefault)
            g.Use(middleware.Auth())
            //g.Use(middleware.Auth(middleware.WithExtraVerify(extraVerifyFn))) // add extra verify function
        }
    
        // Case 2: custom jwt options, signKey, signMethod(HS512), expiry time(48 hour), fields, claims
        {
            r.POST("/auth/login", LoginCustom)
            signKey := []byte("your-sign-key")
            g.Use(middleware.Auth(middleware.WithSignKey(signKey)))
            //g.Use(middleware.Auth(middleware.WithSignKey(signKey), middleware.WithExtraVerify(extraVerifyFn))) // add extra verify function
        }
    
        g.GET("/user/:id", GetByID)
        //g.PUT("/user/:id", Create)
        //g.DELETE("/user/:id", DeleteByID)
    
        r.Run(":8080")
    }
    
    func customGenerateToken(uid string, fields map[string]interface{}) (string, error) {
        _, token, err := jwt.GenerateToken(
            uid,
            jwt.WithGenerateTokenSignKey([]byte("custom-sign-key")),
            jwt.WithGenerateTokenSignMethod(jwt.HS512),
            jwt.WithGenerateTokenFields(fields),
            jwt.WithGenerateTokenClaims([]jwt.RegisteredClaimsOption{
                jwt.WithExpires(time.Hour * 48),
                //jwt.WithIssuedAt(now),
                // jwt.WithSubject("123"),
                // jwt.WithIssuer("https://middleware.example.com"),
                // jwt.WithAudience("https://api.example.com"),
                // jwt.WithNotBefore(now),
                // jwt.WithJwtID("abc1234xxx"),
            }...),
        )
    
        return token, err
    }
    
    func LoginDefault(c *gin.Context) {
        // ......
    
        _, token, err := jwt.GenerateToken("100")
    
        response.Success(c, token)
    }
    
    func LoginCustom(c *gin.Context) {
        // ......
    
        uid := "100"
        fields := map[string]interface{}{
            "name":   "bob",
            "age":    10,
            "is_vip": true,
        }
    
        token, err := customGenerateToken(uid, fields)
    
        response.Success(c, token)
    }
    
    func GetByID(c *gin.Context) {
        uid := c.Param("id")
    
        // if necessary, claims can be got from gin context.
        claims, ok := middleware.GetClaims(c)
        //uid := claims.UID
        //name, _ := claims.GetString("name")
        //age, _ := claims.GetInt("age")
        //isVip, _ := claims.GetBool("is_vip")
    
        response.Success(c, gin.H{"id": uid})
    }
    
    func extraVerifyFn(claims *jwt.Claims, c *gin.Context) error {
        // check if token is about to expire (less than 10 minutes remaining)
        if time.Now().Unix()-claims.ExpiresAt.Unix() < int64(time.Minute*10) {
            token, err := claims.NewToken(time.Hour*24, jwt.HS512, []byte("your-sign-key")) // same signature as jwt.GenerateToken
            if err != nil {
                return err
            }
            c.Header("X-Renewed-Token", token)
        }
    
        // judge whether the user is disabled, query whether jwt id exists from the blacklist
        //if CheckBlackList(uid, claims.ID) {
        //    return errors.New("user is disabled")
        //}
    
        return nil
    }
    ```

<br>

### Tracing middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
    "github.com/Eric-Guo/sponge/pkg/tracer"
)

func InitTrace(serviceName string) {
    exporter, err := tracer.NewJaegerAgentExporter("192.168.3.37", "6831")
    if err != nil {
        panic(err)
    }

    resource := tracer.NewResource(
        tracer.WithServiceName(serviceName),
        tracer.WithEnvironment("dev"),
        tracer.WithServiceVersion("demo"),
    )

    tracer.Init(exporter, resource) // collect all by default
}

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    r.Use(middleware.Tracing("your-service-name"))

    // ......
    return r
}

// if necessary, you can create a span in the program
func CreateSpanDemo(serviceName string, spanName string, ctx context.Context) {
    _, span := otel.Tracer(serviceName).Start(
        ctx, spanName,
        trace.WithAttributes(attribute.String(spanName, time.Now().String())),
    )
    defer span.End()

    // ......
}
```

<br>

### Metrics middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware/metrics"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    r.Use(metrics.Metrics(r,
        //metrics.WithMetricsPath("/demo/metrics"), // default is /metrics
        metrics.WithIgnoreStatusCodes(http.StatusNotFound), // ignore status codes
        //metrics.WithIgnoreRequestMethods(http.MethodHead),  // ignore request methods
        //metrics.WithIgnoreRequestPaths("/ping", "/health"), // ignore request paths
    ))

    // ......
    return r
```

<br>

### Request id middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    // Case 1: default request id
    {
        r.Use(middleware.RequestID())
    }
    // Case 2: custom request id key
    {
        //r.User(middleware.RequestID(
        //    middleware.WithContextRequestIDKey("your ctx request id key"), // default is request_id
        //    middleware.WithHeaderRequestIDKey("your header request id key"), // default is X-Request-Id
        //))
        // If you change the ContextRequestIDKey, you have to set the same key name if you want to print the request id in the mysql logs as well.
        // example:
        //     db, err := mysql.Init(dsn,mysql.WithLogRequestIDKey("your ctx request id key"))  // print request_id
    }

    // ......
    return r
}
```

<br>

### Timeout middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/gin/middleware"
)

func NewRouter() *gin.Engine {
    r := gin.Default()
    // ......

    // Case 1: global set timeout
    {
        r.Use(middleware.Timeout(time.Second*5))
    }
    // Case 2: set timeout for specifyed router
    {
        r.GET("/userExample/:id", middleware.Timeout(time.Second*3), GetByID)
    }
    // Note: If timeout is set both globally and in the router, the minimum timeout prevails

    // ......
    return r
}

func GetByID(c *gin.Context) {
    // do something
}
```
