## app

Start and stop services gracefully, using [errgroup](golang.org/x/sync/errgroup) to ensure that multiple services are started properly at the same time.

<br>

### Example of use

```go
import "github.com/go-dev-frame/sponge/pkg/app"

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
