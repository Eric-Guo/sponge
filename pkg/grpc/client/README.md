## gRPC client

`client` is a gRPC client library for Go. It provides a simple way to connect to a gRPC server and call its methods.

### Example of use

```go
package main

import (
    "context"
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/grpc/client"
    pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func main() {
    conn, err := client.NewClient("127.0.0.1:8282",
        //client.WithServiceDiscover(getDiscovery(), false),
        //client.WithLoadBalance(),
        //client.WithSecure(credentials),
        //client.WithUnaryInterceptor(unaryInterceptors...),
        //client.WithStreamInterceptor(streamInterceptors...),
    )
    if err != nil {
        panic(err)
    }

    greeterClient := pb.NewGreeterClient(conn)
    reply, err := greeterClient.SayHello(context.Background(), &pb.HelloRequest{Name: "Alice"})
    if err != nil {
        panic(err)
    }
    fmt.Printf("Greeting: %s\n", reply.GetMessage())

    conn.Close()
}
```
