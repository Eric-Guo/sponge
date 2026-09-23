// Package main is the http and grpc server of the application.
package main

import (
	"github.com/Eric-Guo/sponge/pkg/app"

	"github.com/Eric-Guo/sponge/cmd/serverNameExample_grpcHttpPbExample/initial"
)

func main() {
	initial.InitApp()
	services := initial.CreateServices()
	closes := initial.Close(services)

	a := app.New(services, closes)
	a.Run()
}
