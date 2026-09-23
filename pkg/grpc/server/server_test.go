package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Eric-Guo/sponge/pkg/grpc/metrics"
	"github.com/Eric-Guo/sponge/pkg/logger"
	"github.com/Eric-Guo/sponge/pkg/utils"
)

var fn = func(s *grpc.Server) {
	// pb.RegisterGreeterServer(s, &greeterServer{})
}

var srFn = func() error {
	//iRegistry, instance, err := consul.NewRegistry(cfg.Consul.Addr, id, cfg.App.Name, []string{instanceEndpoint})
	//if err != nil {
	//	return err
	//}
	// return iRegistry.Register(ctx, instance)
	return nil
}

var unaryInterceptors = []grpc.UnaryServerInterceptor{
	func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		return nil, nil
	},
}

var streamInterceptors = []grpc.StreamServerInterceptor{
	func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return nil
	},
}

func TestRun(t *testing.T) {
	port, _ := utils.GetAvailablePort()
	srv, _ := Run(port, fn,
		WithSecure(insecure.NewCredentials()),
		WithUnaryInterceptor(unaryInterceptors...),
		WithStreamInterceptor(streamInterceptors...),
		WithServiceRegister(srFn),
		WithStatConnections(metrics.WithConnectionsLogger(logger.Get()), metrics.WithConnectionsGauge()),
	)
	defer srv.Stop()
	t.Log("grpc server started", port)
	time.Sleep(time.Second * 2)

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(time.Second * 2)
	_ = conn.Close()
	time.Sleep(time.Second * 1)
}
