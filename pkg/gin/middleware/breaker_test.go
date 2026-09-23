package middleware

import (
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Eric-Guo/sponge/pkg/container/group"
	"github.com/Eric-Guo/sponge/pkg/gin/response"
	"github.com/Eric-Guo/sponge/pkg/httpcli"
	"github.com/Eric-Guo/sponge/pkg/shield/circuitbreaker"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

func runCircuitBreakerHTTPServer() string {
	serverAddr, requestAddr := utils.GetLocalHTTPAddrPairs()

	degradeHandler := func(c *gin.Context) {
		response.Output(c, http.StatusOK, "degrade")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(CircuitBreaker(
		WithGroup(group.NewGroup(func() interface{} {
			return circuitbreaker.NewBreaker()
		})),
		WithBreakerOption(
			circuitbreaker.WithSuccess(75),           // default 60
			circuitbreaker.WithRequest(200),          // default 100
			circuitbreaker.WithBucket(20),            // default 10
			circuitbreaker.WithWindow(time.Second*5), // default 3s
		),
		WithValidCode(http.StatusForbidden),
		WithDegradeHandler(degradeHandler),
	))

	r.GET("/hello", func(c *gin.Context) {
		if rand.Int()%2 == 0 {
			response.Output(c, http.StatusInternalServerError)
		} else {
			response.Success(c, "localhost"+serverAddr)
		}
	})

	go func() {
		err := r.Run(serverAddr)
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Millisecond * 200)
	return requestAddr
}

func TestCircuitBreaker(t *testing.T) {
	requestAddr := runCircuitBreakerHTTPServer()

	var success, failures, degradeCount int32
	for j := 0; j < 5; j++ {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				result := &httpcli.StdResult{}
				err := httpcli.Get(result, requestAddr+"/hello")
				if err != nil {
					//if errors.Is(err, ErrNotAllowed) {
					//	atomic.AddInt32(&countBreaker, 1)
					//}
					atomic.AddInt32(&failures, 1)
					continue
				}
				if result.Data == "degrade" {
					atomic.AddInt32(&degradeCount, 1)
				} else {
					atomic.AddInt32(&success, 1)
				}
			}
		}()

		wg.Wait()
		t.Logf("%s   success: %d, failures: %d,  degradeCount: %d\n",
			time.Now().Format(time.RFC3339Nano), success, failures, degradeCount)
	}
}
