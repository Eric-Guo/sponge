package interceptor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Eric-Guo/sponge/pkg/errcode"
	"github.com/Eric-Guo/sponge/pkg/logger"
)

func TestUnaryClientLog(t *testing.T) {
	addr := newUnaryRPCServer()
	time.Sleep(time.Millisecond * 200)
	cli := newUnaryRPCClient(addr,
		UnaryClientRequestID(),
		UnaryClientLog(logger.Get(), WithReplaceGRPCLogger(), WithPrintErrorByCodes(errcode.StatusInvalidParams.Code())),
	)
	_ = sayHelloMethod(context.Background(), cli)
}

func TestUnaryServerLog(t *testing.T) {
	addr := newUnaryRPCServer(
		UnaryServerRequestID(),
		UnaryServerLog(logger.Get(), WithReplaceGRPCLogger()),
	)
	time.Sleep(time.Millisecond * 200)
	cli := newUnaryRPCClient(addr)
	_ = sayHelloMethod(context.Background(), cli)
}

func TestUnaryServerSimpleLog(t *testing.T) {
	addr := newUnaryRPCServer(
		UnaryServerRequestID(),
		UnaryServerSimpleLog(logger.Get(), WithReplaceGRPCLogger()),
	)
	time.Sleep(time.Millisecond * 200)
	cli := newUnaryRPCClient(addr)
	_ = sayHelloMethod(context.Background(), cli)
}

func TestStreamClientLog(t *testing.T) {
	addr := newStreamRPCServer()
	time.Sleep(time.Millisecond * 200)
	cli := newStreamRPCClient(addr,
		StreamClientRequestID(),
		StreamClientLog(logger.Get(), WithReplaceGRPCLogger()),
	)
	_ = discussHelloMethod(context.Background(), cli)
	time.Sleep(time.Millisecond)
}

func TestUnaryServerLog_ignore(t *testing.T) {
	addr := newUnaryRPCServer(
		UnaryServerLog(logger.Get(),
			WithMaxLen(200),
			WithMarshalFn(func(reply interface{}) []byte {
				data, _ := json.Marshal(reply)
				return data
			}),
			WithLogIgnoreMethods("/proto.Greeter/SayHello"),
		),
	)
	time.Sleep(time.Millisecond * 200)
	cli := newUnaryRPCClient(addr)
	_ = sayHelloMethod(context.Background(), cli)
}

func TestStreamServerLog(t *testing.T) {
	addr := newStreamRPCServer(
		StreamServerRequestID(),
		StreamServerLog(logger.Get(),
			WithReplaceGRPCLogger(),
		),
		StreamServerSimpleLog(logger.Get(),
			WithReplaceGRPCLogger(),
		),
	)
	time.Sleep(time.Millisecond * 200)
	cli := newStreamRPCClient(addr)
	_ = discussHelloMethod(context.Background(), cli)
	time.Sleep(time.Millisecond)
}

// ----------------------------------------------------------------------------------------

func TestNilLog(t *testing.T) {
	UnaryClientLog(nil)
	StreamClientLog(nil)
	UnaryServerLog(nil)
	UnaryServerSimpleLog(nil)
	StreamServerLog(nil)
	StreamServerSimpleLog(nil)
}
