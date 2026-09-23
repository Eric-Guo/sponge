package initial

import (
	"context"
	"time"

	"github.com/Eric-Guo/sponge/pkg/app"
	"github.com/Eric-Guo/sponge/pkg/logger"
	"github.com/Eric-Guo/sponge/pkg/tracer"

	"github.com/Eric-Guo/sponge/internal/config"
	//"github.com/Eric-Guo/sponge/internal/rpcclient"
)

// Close releasing resources after service exit
func Close(servers []app.IServer) []app.Close {
	var closes []app.Close

	// close server
	for _, s := range servers {
		closes = append(closes, s.Stop)
	}

	// close the rpc client connection
	// example:
	//closes = append(closes, func() error {
	//	return rpcclient.CloseServerNameExampleRPCConn()
	//})

	// close tracing
	if config.Get().App.EnableTrace {
		closes = append(closes, func() error {
			ctx, _ := context.WithTimeout(context.Background(), 2*time.Second) //nolint
			return tracer.Close(ctx)
		})
	}

	// close logger
	closes = append(closes, func() error {
		return logger.Sync()
	})

	return closes
}
