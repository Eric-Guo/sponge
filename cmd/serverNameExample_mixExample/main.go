// Package main is the http and grpc server of the application.
package main

import (
	"github.com/Eric-Guo/sponge/pkg/app"

	"github.com/Eric-Guo/sponge/cmd/serverNameExample_mixExample/initial"
)

// @title serverNameExample api docs
// @description http server api docs
// @schemes http https
// @version v1.0.0
// @host localhost:8080
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type Bearer your-jwt-token to Value
func main() {
	initial.InitApp()
	services := initial.CreateServices()
	closes := initial.Close(services)

	a := app.New(services, closes)
	a.Run()
}
