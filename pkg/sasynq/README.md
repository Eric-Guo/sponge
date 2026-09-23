## sasynq

`sasynq` is a wrapper around the excellent [asynq](https://github.com/hibiken/asynq) library. It provides a simpler and more user-friendly SDK while remaining fully compatible with native asynq usage patterns. Its main features include:

- Support for Redis Cluster and Sentinel for high availability and horizontal scalability.
- Distributed task queues with support for priority queues, delayed queues, unique tasks (to prevent duplicate execution), cancel task, and periodic task scheduling.
- Built-in mechanisms for task retries (with customizable retry counts), timeouts, and deadlines.
- Flexible scheduling for immediate, delayed, or specific-time execution.
- Unified logging using zap.

`sasynq` streamlines asynchronous and distributed task processing in Go, helping you write clean and maintainable background job code quickly and safely.

<br>

## Example of use

### Queues

#### Defining Task Payloads and Handlers

Here’s how to define task payloads and handlers in `sasynq`:

```go
// example/common/task.go
package common

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/hibiken/asynq"
    "github.com/Eric-Guo/sponge/pkg/sasynq"
)

// ----------------------------- Definition Method 1 (recommended)----------------------------------

const TypeEmailSend = "email:send"

type EmailPayload struct {
    UserID  int    `json:"user_id"`
    Message string `json:"message"`
}

func HandleEmailTask(ctx context.Context, p *EmailPayload) error {
    fmt.Printf("[Email] Task for UserID %d completed successfully\n", p.UserID)
    return nil
}

// ----------------------------- Definition Method  2 ----------------------------------

const TypeSMSSend = "sms:send"

type SMSPayload struct {
    UserID  int    `json:"user_id"`
    Message string `json:"message"`
}

func (p *SMSPayload) ProcessTask(ctx context.Context, t *asynq.Task) error {
    fmt.Printf("[SMS] Task for UserID %d completed successfully\n", p.UserID)
    return nil
}

// ----------------------------- Definition Method  3 ----------------------------------

const TypeMsgNotification = "msg:notification"

type MsgNotificationPayload struct {
    UserID  int    `json:"user_id"`
    Message string `json:"message"`
}

func HandleMsgNotificationTask(ctx context.Context, t *asynq.Task) error {
    var p MsgNotificationPayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %w", err)
    }
    fmt.Printf("[MSG] Task for UserID %d completed successfully\n", p.UserID)
    return nil
}

const TypeUniqueEmailSend = "unique:email:send"
````

<br>

#### Producer Example

A producer enqueues tasks with various options like priority, delays, deadlines, and unique IDs.

```go
// example/producer/main.go
package main

import (
    "fmt"
    "time"

    "github.com/Eric-Guo/sponge/pkg/sasynq"
    "example/common"
)

func runProducer(client *sasynq.Client) error {
    // Immediate enqueue with critical priority
    userPayload1 := &common.EmailPayload{
        UserID:  101,
        Message: "This is a message that is immediately queued, with critical priority",
    }
    _, info, err := client.EnqueueNow(common.TypeEmailSend, userPayload1,
        sasynq.WithQueue("critical"),
        sasynq.WithRetry(5),
    )
    if err != nil {
        return err
    }
    fmt.Printf("enqueued task: type=%s, id=%s, queue=%s\n", common.TypeEmailSend, info.ID, info.Queue)

    // Enqueue after a 5-second delay
    userPayload2 := &common.SMSPayload{
        UserID:  202,
        Message: "This is a message added to the queue after a 5-second delay, with default priority",
    }
    _, info, err = client.EnqueueIn(5*time.Second, common.TypeSMSSend, userPayload2,
        sasynq.WithQueue("default"),
        sasynq.WithRetry(3),
    )
    if err != nil {
        return err
    }
    fmt.Printf("enqueued task: type=%s, id=%s, queue=%s\n", common.TypeSMSSend, info.ID, info.Queue)

    // Enqueue to run at a specific time
    userPayload3 := &common.MsgNotificationPayload{
        UserID:  303,
        Message: "This is a message scheduled to run at a specific time, with low priority",
    }
    _, info, err = client.EnqueueAt(time.Now().Add(10*time.Second), common.TypeMsgNotification, userPayload3,
        sasynq.WithQueue("low"),
        sasynq.WithRetry(1),
    )
    if err != nil {
        return err
    }
    fmt.Printf("enqueued task: type=%s, id=%s, queue=%s\n", common.TypeMsgNotification, info.ID, info.Queue)

    // Example of using NewTask directly
    userPayload4 := &common.EmailPayload{
        UserID:  404,
        Message: "This is a test message, with low priority, a 15-second deadline, and a unique ID",
    }
    task, err := sasynq.NewTask(common.TypeEmailSend, userPayload4)
    if err != nil {
        return err
    }
    info, err = client.Enqueue(task,
        sasynq.WithQueue("low"),
        sasynq.WithMaxRetry(1),
        sasynq.WithDeadline(time.Now().Add(15*time.Second)),
        sasynq.WithTaskID("unique-id-xxxx-xxxx"),
    )
    if err != nil {
        return err
    }
    fmt.Printf("enqueued task: type=%s, id=%s, queue=%s\n", common.TypeEmailSend, info.ID, info.Queue)

    // Example of using EnqueueUnique
    userPayload5 := &common.EmailPayload{
        UserID:  505,
        Message: "This is a unique task, with default priority, a 1-minute deadline",
    }
    userPayload5 := &EmailPayload{UserID: 505, Message: "unique task"}
    _, info5, err := client.EnqueueUnique(time.Minute, common.TypeUniqueEmailSend, userPayload5,
        sasynq.WithQueue("default"),
        sasynq.WithMaxRetry(3))
    if err != nil {
        return err
    }
    fmt.Printf("enqueued task: type=%s, id=%s, queue=%s\n", common.TypeUniqueEmailSend, info5.ID, info5.Queue)

    return nil
}

func main() {
    cfg := sasynq.RedisConfig{
        Addr: "localhost:6379",
    }
    client := sasynq.NewClient(cfg)

    err := runProducer(client)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    fmt.Println("All tasks enqueued.")
}
```

<br>

#### Consumer Example

A consumer server can register handlers in three different ways:

```go
// example/consumer/main.go
package main

import (
    "github.com/Eric-Guo/sponge/pkg/sasynq"
    "github.com/Eric-Guo/sponge/pkg/logger"
    "example/common"
)

func runConsumer(redisCfg sasynq.RedisConfig) (*sasynq.Server, error) {
    serverCfg := sasynq.DefaultServerConfig(sasynq.WithLogger(logger.Get())) // Uses critical, default, and low queues by default
    srv := sasynq.NewServer(redisCfg, serverCfg)

    // Attach logging middleware
    srv.Use(sasynq.LoggingMiddleware(sasynq.WithLogger(logger.Get())))

    // Register task handlers (three methods available):
    sasynq.RegisterTaskHandler(srv.Mux(), common.TypeEmailSend, sasynq.HandleFunc(common.HandleEmailTask)) // Method 1 (recommended)
    srv.Register(common.TypeSMSSend, &common.SMSPayload{}) // Method 2: register struct as payload
    srv.RegisterFunc(common.TypeMsgNotification, common.HandleMsgNotificationTask) // Method 3: register function directly

    sasynq.RegisterTaskHandler(srv.Mux(), common.TypeUniqueEmailSend, sasynq.HandleFunc(common.HandleEmailTask))
    
    srv.Run()

    return srv, nil
}

func main() {
    cfg := sasynq.RedisConfig{
        Addr: "localhost:6379",
    }
    srv, err := runConsumer(cfg)
    if err != nil {
        panic(err)
    }
    srv.WaitShutdown()
}
```

<br>

### Periodic Tasks

`sasynq` makes scheduling recurring tasks very simple.

```go
package main

import (
    "context"
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/sasynq"
    "github.com/Eric-Guo/sponge/pkg/logger"
)

const TypeScheduledGet = "scheduled:get"

type ScheduledGetPayload struct {
    URL string `json:"url"`
}

func handleScheduledGetTask(ctx context.Context, p *ScheduledGetPayload) error {
    fmt.Printf("[Get] Task for URL %s completed successfully\n", p.URL)
    return nil
}

// -----------------------------------------------------------------------

func registerSchedulerTasks(scheduler *sasynq.Scheduler) error {
    payload1 := &ScheduledGetPayload{URL: "https://google.com"}
    entryID1, err := scheduler.RegisterTask("@every 2s", TypeScheduledGet, payload1)
    if err != nil {
        return err
    }
    fmt.Printf("Registered periodic task with entry ID: %s\n", entryID1)

    payload2 := &ScheduledGetPayload{URL: "https://bing.com"}
    entryID2, err := scheduler.RegisterTask("@every 3s", TypeScheduledGet, payload2)
    if err != nil {
        return err
    }
    fmt.Printf("Registered periodic task with entry ID: %s\n", entryID2)

    scheduler.Run()

    return nil
}

func runServer(redisCfg sasynq.RedisConfig) (*sasynq.Server, error) {
    serverCfg := sasynq.DefaultServerConfig(sasynq.WithLogger(logger.Get()))
    srv := sasynq.NewServer(redisCfg, serverCfg)
    srv.Use(sasynq.LoggingMiddleware())

    // Register handler for scheduled tasks
    sasynq.RegisterTaskHandler(srv.Mux(), TypeScheduledGet, sasynq.HandleFunc(handleScheduledGetTask))

    srv.Run()

    return srv, nil
}

func main() {
    cfg := sasynq.RedisConfig{
        Addr: "localhost:6379",
    }

    scheduler := sasynq.NewScheduler(cfg, sasynq.WithSchedulerLogger(sasynq.WithLogger(logger.Get())))
    err := registerSchedulerTasks(scheduler)
    if err != nil {
        panic(err)
    }

    srv, err := runServer(cfg)
    if err != nil {
        panic(err)
    }
    srv.Shutdown()
}
```

<br>

### Cancel Tasks

1. For one-time tasks, the `inspector.CancelTask(queue, taskID)` function can be used to cancel the task. The example code is as follows:


```go
package main

import (
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/sasynq"
)

var inspector = sasynq.NewInspector(sasynq.DefaultServerConfig())

func main() {
    queue := "default"
    taskID := "task-id-xxxx-xxxx"
    isScheduled := false // set to true for scheduled tasks

    err := cancelTask(queue, taskID, isScheduled)
    if err != nil {
        fmt.Printf("Failed to cancel task: %v\n", err)
    }
}

func cancelTask(queue string, taskID string, isScheduled bool) error{
    var err error
    if isScheduled {
        err = inspector.CancelTask(queue, taskID)
    } else {
        err = inspector.CancelTask("", taskID) // queue is empty string for non-scheduled tasks
    }
    if err != nil {
        return err
    }

    return nil
}
```

<br>

2. For periodic scheduled tasks, the `scheduler.Unregister(entryID)` function can be used to cancel scheduled tasks. The example code is as follows:

```go
package main

import (
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/sasynq"
)

var scheduler = sasynq.NewScheduler(sasynq.DefaultServerConfig())

func main() {
    entryID := "entry-id-xxxx-xxxx" // scheduler.RegisterTask() returns this ID

    err := scheduler.Unregister(entryID)
    if err != nil {
        fmt.Printf("Failed to unregister periodic scheduled tasks: %v\n", err)
    }
}
```

<br>
