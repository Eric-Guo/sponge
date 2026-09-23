## SSE

A high-performance Go language Server-Sent Events (SSE) server and client implementation, supporting uni-cast and broadcast events, automatic reconnection, message persistence, and other features.

### Features

  - 🚀 High-performance Event Hub for managing client connections
  - 🔌 Supports automatic reconnection and event retransmission on disconnect
  - 📊 Built-in push statistics and performance monitoring
  - 🔒 Thread-safe client management
  - ⏱️ Supports timeout retries and asynchronous task processing
  - 💾 Optional persistent storage interface
  - ❤️ Built-in heartbeat detection mechanism

<br>

### Example of Use

#### Server Example

```go
package main

import (
    "net/http"
    "strconv"
    "time"
    "math/rand"
    "github.com/gin-gonic/gin"
    "github.com/Eric-Guo/sponge/pkg/sse"
)

func main() {
    // Initialize SSE Hub
    hub := sse.NewHub()
    defer hub.Close()

    // Create Gin router
    r := gin.Default()

    // SSE Event Stream Interface, requires authentication to set uid
    r.GET("/events", func(c *gin.Context) {
        var uid string
        u, isExists := c.Get("uid")
        if !isExists {
            uid = strconv.Itoa(rand.Intn(99) + 100) // mock uid
        } else {
            uid, _ = u.(string)
        }
        hub.Serve(c, uid)
    })

    // Register event push endpoint, supports pushing to specified users and broadcast pushing
    // Push to specified users
    // curl -X POST -H "Content-Type: application/json" -d '{"uids":["u001"],"events":[{"event":"message","data":"hello_world"}]}' http://localhost:8080/push
    // Broadcast push, not specifying users means pushing to all users
    // curl -X POST -H "Content-Type: application/json" -d '{"events":[{"event":"message","data":"hello_world"}]}' http://localhost:8080/push
    r.POST("/push", hub.PushEventHandler())

    // simulated event push
    go func() {
        i := 0
        for {
            time.Sleep(time.Second * 5)
            i++
            e := &sse.Event{Event: sse.DefaultEventType, Data: "hello_world_" + strconv.Itoa(i)}
            _ = hub.Push(nil, e) // broadcast push
            //_ = hub.Push([]string{uid}, e) // specified user push
        }
    }()

    // Start HTTP server
    if err := http.ListenAndServe(":8080", r); err != nil {
        panic(err)
    }
}
```

<br>

#### Client Example

```go
package main

import (
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/sse"
)

func main() {
    url := "http://localhost:8080/events"

    // Create SSE client
    client := sse.NewClient(url)

    client.OnEvent(sse.DefaultEventType, func(event *sse.Event) {
        fmt.Printf("Received: %#v\n", event)
    })

    err := client.Connect()
    if err != nil {
        fmt.Printf("Connection failed: %v\n", err)
        return
    }

    fmt.Println("SSE client started, press Ctrl+C to exit")
    <-client.Wait()
}
```

<br>

### Advanced Configuration

#### Using Persistent Storage

You can implement map, redis, mysql and other storage to achieve persistent storage and query of events. Example code:

```go
// Implement the Store interface
type MyStore struct{}

func (s *MyStore) Save(ctx context.Context, e *sse.Event) error {
    // Implement event storage logic
    return nil
}

func (s *MyStore) ListByLastID(ctx context.Context, eventType string, lastID string, pageSize int) ([]*sse.Event, string, error) {
    // Implement event query logic, paginate query, return event list, last event ID
    return nil, nil
}

// Create Hub with storage
hub := sse.NewHub(sse.WithStore(&MyStore{}))
```

<br>

#### Configure whether events need to be resent when the client disconnects and reconnects

To enable this feature, it needs to be used with event persistent storage. Example code:

```go
hub := sse.NewHub(
    sse.WithStore(&MyStore{}),
    sse.WithEnableResendEvents(),
)
```

<br>

#### Customizing Push Failed Event Handling

Code example:

```go
fn := func(uid string, event *sse.Event) {
    // Custom handling logic for push failures, such as logging or saving to database
    log.Printf("Push failed: User %s, Event ID %s", uid, event.ID)
}

// Create Hub with push failed handling
hub := sse.NewHub(sse.WithPushFailedHandleFn(fn))
```

<br>

### API Reference

#### Hub Methods

  - `NewHub(opts ...HubOption) *Hub`: Creates a new event hub, supporting custom persistence, re-sending events, logging, push event buffer size, and concurrent push event goroutine options.
  - `Push(uids []string, events ...*Event) error`: Pushes events to specified users or all users
  - `OnlineClientsNum() int`: Gets the number of online clients
  - `Close()`: Closes the event hub
  - `PrintPushStats()`: Prints push statistics

<br>

#### Serve Method

  - `Serve(c *gin.Context, hub *Hub, uid string, opts...ServeOption)`: Handles SSE client connection requests, supports setting custom request headers.

<br>

#### Client Methods

  - `NewClient(url string) *SSEClient`: Creates a new SSE client, supporting custom request headers, reconnection interval, and log options.
  - `Connect() error`: Connects to the server
  - `Disconnect()`: Disconnects
  - `OnEvent(eventType string, callback EventCallback)`: Registers an event callback

<br>

### Performance Tuning

  - `WithChanBufferSize(size int)`: Sets the broadcast channel buffer size
  - `WithWorkerNum(num int)`: Sets the number of asynchronous worker goroutines
