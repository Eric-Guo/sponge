## app

Start and stop services gracefully, using [errgroup](golang.org/x/sync/errgroup) to ensure that multiple services are started properly at the same time.

<br>

### Example of use

```go
import "github.com/Eric-Guo/sponge/pkg/app"

func main() {
    initApp()
    services := CreateServices()
    closes := Close(services)

    a := app.New(services, closes)
    a.Run()
}

func initApp() {
    // get configuration

    // initializing log

    // initializing database

    // ......
}

func CreateServices() []app.IServer {
    var servers []app.IServer

    // create an HTTP service
    httpAddr := ":8080" // or get from configuration
    httpServer := server.NewHTTPServer(
        httpAddr,
        server.WithHTTPIsProd(true), // run in release mode
    )
    servers = append(servers, httpServer)

    // create a gRPC service (optional)
    // grpcServer := server.NewGRPCServer(
    //
    // )
    // servers = append(servers, grpcServer)

    return servers
}

func Close(servers []app.IServer) []app.Close {
    var closes []app.Close

    // close servers
    for _, s := range servers {
        closes = append(closes, s.Stop)
    }

    // close other resources (database, logger, tracing, etc.)
    closes = append(closes, func() error {
        // TODO: call db.Close()
        return nil
    })

    return closes
}
```


### Local upstream processes

`NewUpstreamServer(UpstreamConfig{...})` implements `IServer` for a local command.
It supports quoted arguments, a working directory, extra environment variables,
a target `PORT`, and a configurable shutdown signal. Add the supervisor to the
services passed to `app.New`; initialize the application logger before starting
services. A UNIX socket setting takes precedence over exporting the target port.
### Supervising an upstream application

Applications hosting a supervised Rails/Puma process can use
`os.Exit(app.New(servers, closes).RunWithExitCode())`. This additive lifecycle
method shuts down on any service completion, including a successful child exit,
and runs every closer before returning. Put the upstream closer first so signals
reach it before HTTP connections drain. Parent termination signals override the
upstream's configured StopSignal; an upstream killed by a signal returns
`128 + signal`, while a child that handles the signal and exits normally retains
its chosen exit code. `UpstreamServer.ExitCode()` exposes the final status.
The existing `Run()` behavior remains available for other Sponge applications.
