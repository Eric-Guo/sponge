## grpc server

`server` is a gRPC server library for Go, it provides a simple way to create and run a gRPC server.

### Example of use

```go
package main

import (
    "context"
    "fmt"
    "github.com/Eric-Guo/sponge/pkg/grpc/server"
    "google.golang.org/grpc"
    pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

type greeterServer struct {
    pb.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    fmt.Printf("Received: %v\n", in.GetName())
    return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
    port := 8282
    registerFn := func(s *grpc.Server) {
        pb.RegisterGreeterServer(s, &greeterServer{})
        // Register other services here
    }

    fmt.Printf("Starting server on port %d\n", port)
    srv, err := server.Run(port, registerFn,
        //server.WithSecure(credentials),
        //server.WithUnaryInterceptor(unaryInterceptors...),
        //server.WithStreamInterceptor(streamInterceptors...),
        //server.WithServiceRegister(srFn), // register service address to Consul/Etcd/Zookeeper...
        //server.WithStatConnections(metrics.WithConnectionsLogger(logger.Get()), metrics.WithConnectionsGauge()),
    )
    if err != nil {
        panic(err)
    }
    defer srv.Stop()

    select {}
}
```
