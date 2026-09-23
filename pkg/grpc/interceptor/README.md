## interceptor

Common interceptors for gRPC server and client side, including:

- [Logging](README.md#logging-interceptor)
- [Recovery](README.md#recovery-interceptor)
- [Retry](README.md#retry-interceptor)
- [Rate limiter](README.md#rate-limiter-interceptor)
- [Circuit breaker](README.md#circuit-breaker-interceptor)
- [Timeout](README.md#timeout-interceptor)
- [Tracing](README.md#tracing-interceptor)
- [Request id](README.md#request-id-interceptor)
- [Metrics](README.md#metrics-interceptor)
- [JWT authentication](README.md#jwt-authentication-interceptor)

<br>

### Example of use

All import paths are "github.com/Eric-Guo/sponge/pkg/grpc/interceptor".

#### Logging interceptor

**gRPC server side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "github.com/Eric-Guo/sponge/pkg/logger"
    "google.golang.org/grpc"
)

func setServerOptions() []grpc.ServerOption {
    var options []grpc.ServerOption

    option := grpc.ChainUnaryInterceptor(
        // if you don't want to log reply data, you can use interceptor.StreamServerSimpleLog instead of interceptor.UnaryServerLog,
        interceptor.UnaryServerLog( // set unary server logging
            logger.Get(),
            interceptor.WithReplaceGRPCLogger(),
            //interceptor.WithMarshalFn(fn), // customised marshal function, default is jsonpb.Marshal
            //interceptor.WithLogIgnoreMethods(fullMethodNames), // ignore methods logging
            //interceptor.WithMaxLen(400), // logging max length, default 300
        ),
    )
    options = append(options, option)

    return options
}


// you can also set stream server logging
```

**gRPC client side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    option := grpc.WithChainUnaryInterceptor(
        interceptor.UnaryClientLog( // set unary client logging
            logger.Get(),
            interceptor.WithReplaceGRPCLogger(),
        ),
    )
    options = append(options, option)

    return options
}

// you can also set stream client logging
```

<br>

#### Recovery interceptor

**gRPC server side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setServerOptions() []grpc.ServerOption {
    var options []grpc.ServerOption

    option := grpc.ChainUnaryInterceptor(
        interceptor.UnaryServerRecovery(),
    )
    options = append(options, option)

    return options
}
```

**gRPC client side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    option := grpc.WithChainUnaryInterceptor(
        interceptor.UnaryClientRecovery(),
    )
    options = append(options, option)

    return options
}
```

<br>

#### Retry interceptor

**gRPC client side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    // use insecure transfer
    options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))

    // retry
    option := grpc.WithChainUnaryInterceptor(
        interceptor.UnaryClientRetry(
            //interceptor.WithRetryTimes(5), // modify the default number of retries to 3 by default
            //interceptor.WithRetryInterval(100*time.Millisecond), // modify the default retry interval, default 50 milliseconds
            //interceptor.WithRetryErrCodes(), // add trigger retry error code, default is codes.Internal, codes.DeadlineExceeded, codes.Unavailable
        ),
    )
    options = append(options, option)

    return options
}
```

<br>

#### Rate limiter interceptor

Adaptive flow limitation based on hardware resources.

**gRPC server side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    // use insecure transfer
    options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))

    // rate limiter
    option := grpc.ChainUnaryInterceptor(
        interceptor.UnaryServerRateLimit(
            //interceptor.WithWindow(time.Second*5),
            //interceptor.WithBucket(200),
            //interceptor.WithCPUThreshold(600),
            //interceptor.WithCPUQuota(0),
        ),
    )
    options = append(options, option)

    return options
}
```

<br>

#### Circuit breaker interceptor

**gRPC server side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    // use insecure transfer
    options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))

    // circuit breaker
    option := grpc.ChainUnaryInterceptor(
        interceptor.UnaryServerCircuitBreaker(
            //interceptor.WithValidCode(codes.DeadlineExceeded), // add error code for circuit breaker
            //interceptor.WithUnaryServerDegradeHandler(handler), // add custom degrade handler
            //interceptor.WithBreakerOption(
                //circuitbreaker.WithSuccess(75),           // default 60
                //circuitbreaker.WithRequest(200),          // default 100
                //circuitbreaker.WithBucket(20),            // default 10
                //circuitbreaker.WithWindow(time.Second*5), // default 3s
            //),
        ),
    )
    options = append(options, option)

    return options
}
```

<br>

#### Timeout interceptor

**gRPC client side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    // use insecure transfer
    options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))

    option := grpc.WithChainUnaryInterceptor(
        interceptor.UnaryClientTimeout(time.Second), // set timeout
    )
    options = append(options, option)

    return options
}
```

<br>

#### Tracing interceptor

**Initialize tracing**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/tracer"
    "go.opentelemetry.io/otel"
)

// initialize tracing
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

// if necessary, you can create a span in the program
func SpanDemo(serviceName string, spanName string, ctx context.Context) {
_, span := otel.Tracer(serviceName).Start(
ctx, spanName,
trace.WithAttributes(attribute.String(spanName, time.Now().String())), // customised attributes
)
defer span.End()

// ......
}
```

**gRPC server side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setServerOptions() []grpc.ServerOption {
    var options []grpc.ServerOption

    // use tracing
    option := grpc.UnaryInterceptor(
        interceptor.UnaryServerTracing(),
    )
    options = append(options, option)

    return options
}
```

**gRPC client side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    // use insecure transfer
    options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))

    // use tracing
    option := grpc.WithUnaryInterceptor(
        interceptor.UnaryClientTracing(),
    )
    options = append(options, option)

    return options
}
```

<br>

#### Metrics interceptor

Click to view [metrics examples](../metrics/README.md).

<br>

#### Request id interceptor

**gRPC server side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setServerOptions() []grpc.ServerOption {
    var options []grpc.ServerOption

    option := grpc.ChainUnaryInterceptor(
        interceptor.UnaryServerRequestID(),
    )
    options = append(options, option)

    return options
}
```

<br>

**gRPC client side**

```go
import (
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "google.golang.org/grpc"
)

func setDialOptions() []grpc.DialOption {
    var options []grpc.DialOption

    // use insecure transfer
    options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))

    option := grpc.WithChainUnaryInterceptor(
        interceptor.UnaryClientRequestID(),
    )
    options = append(options, option)

    return options
}
```

<br>

#### JWT authentication interceptor

**gRPC server side**

```go
package main

import (
    "context"
    "net"
    "time"
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    "github.com/Eric-Guo/sponge/pkg/jwt"
    "google.golang.org/grpc"
    userV1 "user/api/user/v1"
)

func main() {
    list, err := net.Listen("tcp", ":8282")
    server := grpc.NewServer(getUnaryServerOptions()...)
    userV1.RegisterUserServer(server, &user{})
    server.Serve(list)
    select {}
}

func getUnaryServerOptions() []grpc.ServerOption {
    var options []grpc.ServerOption

    // Case1: default options
    {
        options = append(options, grpc.UnaryInterceptor(
            interceptor.UnaryServerJwtAuth(),
        ))
    }

    // Case 2: custom options, signKey, extra verify function, rpc method
    {
        options = append(options, grpc.UnaryInterceptor(
            interceptor.UnaryServerJwtAuth(
                interceptor.WithSignKey([]byte("your_secret_key")),
                interceptor.WithExtraVerify(extraVerifyFn),
                interceptor.WithAuthIgnoreMethods(// specify the gRPC API to ignore token verification(full path)
                    "/api.user.v1.User/Register",
                    "/api.user.v1.User/Login",
                ),
            ),
        ))
    }

    return options
}

type user struct {
    userV1.UnimplementedUserServer
}

// Login ...
func (s *user) Login(ctx context.Context, req *userV1.LoginRequest) (*userV1.LoginReply, error) {
    // check user and password success

    uid := "100"
    fields := map[string]interface{}{"name":   "bob","age":    10,"is_vip": true}

    // Case 1: default jwt options, signKey, signMethod(HS256), expiry time(24 hour)
    {
        _, token, err := jwt.GenerateToken("100")
    }

    // Case 2: custom jwt options, signKey, signMethod(HS512), expiry time(12 hour), fields, claims
    {
        _, token, err := jwt.GenerateToken(
            uid,
            jwt.WithGenerateTokenSignKey([]byte("your_secret_key")),
            jwt.WithGenerateTokenSignMethod(jwt.HS384),
            jwt.WithGenerateTokenFields(fields),
            jwt.WithGenerateTokenClaims([]jwt.RegisteredClaimsOption{
                jwt.WithExpires(time.Hour * 12),
                // jwt.WithIssuedAt(now),
                // jwt.WithSubject("123"),
                // jwt.WithIssuer("https://auth.example.com"),
                // jwt.WithAudience("https://api.example.com"),
                // jwt.WithNotBefore(now),
                // jwt.WithJwtID("abc1234xxx"),
            }...),
        )
    }

    return &userV1.LoginReply{Token: token}, nil
}

func extraVerifyFn(ctx context.Context, claims *jwt.Claims) error {
    // judge whether the user is disabled, query whether jwt id exists from the blacklist
    //if CheckBlackList(uid, claims.ID) {
    //    return errors.New("user is disabled")
    //}

    // get fields from claims
    //uid := claims.UID
    //name, _ := claims.GetString("name")
    //age, _ := claims.GetInt("age")
    //isVip, _ := claims.GetBool("is_vip")

    return nil
}

// GetByID ...
func (s *user) GetByID(ctx context.Context, req *userV1.GetByIDRequest) (*userV1.GetByIDReply, error) {
    // ......

	claims,ok := interceptor.GetJwtClaims(ctx) // if necessary, claims can be got from gin context.

	// ......
}
```

**gRPC client side**

```go
package main

import (
    "context"
    "github.com/Eric-Guo/sponge/pkg/grpc/grpccli"
    "github.com/Eric-Guo/sponge/pkg/grpc/interceptor"
    userV1 "user/api/user/v1"
)

func main() {
    conn, _ := grpccli.NewClient("127.0.0.1:8282")
    cli := userV1.NewUserClient(conn)

    uid := "100"
    ctx := context.Background()

    // Case 1: get authorization from header key is "authorization", value is "Bearer xxx"
    {
        ctx = interceptor.SetAuthToCtx(ctx, authorization)
    }
    // Case 2: get token from grpc server response result
    {
        ctx = interceptor.SetJwtTokenToCtx(ctx, token)
    }

    cli.GetByID(ctx, &userV1.GetUserByIDRequest{Id: 100})
}
```

<br>
