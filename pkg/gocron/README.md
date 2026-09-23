## gocron

Scheduled task library encapsulated on [cron v3](github.com/robfig/cron).

<br>

### Example of use

#### Local Scheduled Tasks

Local scheduled tasks are suitable for single-machine environments, typically used to perform periodic or delayed background jobs, such as data cleanup, log archiving, and local cache refreshing. Example usage:

```go
package main

import (
	"fmt"
	"time"

	"github.com/Eric-Guo/sponge/pkg/gocron"
	"github.com/Eric-Guo/sponge/pkg/logger"
)

var task1 = func() {
	fmt.Println("this is task1")
	fmt.Println("running task list:", gocron.GetRunningTasks())
}

var taskOnce = func() {
	fmt.Println("this is task2, only run once")
	fmt.Println("running task list:", gocron.GetRunningTasks())
}

func main() {
	err := gocron.Init(
			gocron.WithLogger(logger.Get()),
			// gocron.WithLogger(logger.Get(), true), // only print error logs, ignore info logs
		)
	if err != nil {
		panic(err)
	}

	gocron.Run([]*gocron.Task{
		{
			Name:     "task1",
			TimeSpec: "@every 2s",
			Fn:       task1,
		},
		{
			Name:      "taskOnce",
			TimeSpec:  "@every 3s",
			Fn:        taskOnce,
			IsRunOnce: true, // run only once
		},
	}...)

	time.Sleep(time.Second * 10)

	// stop task1
	gocron.DeleteTask("task1")

	// view running tasks
	fmt.Println("running task list:", gocron.GetRunningTasks())
}
```

<br>

#### Distributed Scheduled Tasks

Distributed scheduled tasks are designed for cluster environments, ensuring coordinated task execution across multiple nodes to avoid duplicate scheduling while improving reliability and scalability. Example usage:

[https://github.com/Eric-Guo/sponge/tree/thruster_generate/pkg/sasynq#periodic-tasks](https://github.com/Eric-Guo/sponge/tree/thruster_generate/pkg/sasynq#periodic-tasks)
