## Go Process Kill Utility

A simple, cross-platform Go package for terminating processes by their PID. It prioritizes a graceful shutdown before resorting to a force kill, ensuring that applications have a chance to clean up resources.

### Features

-   **Cross-Platform:** Works seamlessly on Windows, Linux, and macOS.
-   **Graceful Shutdown First:** Always attempts to terminate a process gracefully before forcing it to exit.
-   **Automatic Fallback:** If a graceful shutdown fails or the process does not exit within a timeout period (5 seconds), it automatically performs a force kill.
-   **Simple API:** A single `Kill(pid)` function makes it incredibly easy to use.

<br>

### Usage

Here is a simple example of how to start a process and then terminate it using this package.

```go
package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/Eric-Guo/sponge/pkg/process"
)

func main() {
	var cmd *exec.Cmd

	// Start a long-running process appropriate for the OS.
	if runtime.GOOS == "windows" {
		cmd = exec.Command("timeout", "/t", "30")
	} else {
		cmd = exec.Command("sleep", "30")
	}

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	pid := cmd.Process.Pid
	fmt.Printf("Started process with PID: %d\n", pid)

	// Give the process a moment to initialize.
	time.Sleep(1 * time.Second)
	fmt.Printf("Attempting to kill process %d...\n", pid)

	// Use the Kill function to terminate it.
	if err = process.Kill(pid); err != nil {
		log.Fatalf("Failed to kill process: %v", err)
	}

	fmt.Printf("Successfully terminated process %d.\n", pid)

	// The cmd.Wait() call will now return an error because the process was killed.
	err = cmd.Wait()
	fmt.Printf("Process wait result: %v\n", err)
}
```
