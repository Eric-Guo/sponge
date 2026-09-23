package consul

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"

	"github.com/Eric-Guo/sponge/pkg/servicerd/registry"
)

func TestNewRegistry(t *testing.T) {
	consulAddr := "192.168.3.37:8500"
	id := "serverName_192.168.3.37"
	instanceName := "serverName"
	instanceEndpoints := []string{"grpc://192.168.3.27:8282"}

	// example 1
	iRegistry, serviceInstance, err := NewRegistry(consulAddr, id, instanceName, instanceEndpoints)
	assert.NoError(t, err)
	t.Log(iRegistry, serviceInstance)

	// example 2
	iRegistry, serviceInstance, err = NewRegistryWithOptions(consulAddr, id, instanceName, instanceEndpoints,
		WithHealthCheck(true),
		//consulcli.WithScheme("https"),
		//consulcli.WithDatacenter("foobar"),
		//consulcli.WithToken("xxx"),
		//consulcli.WithWaitTime(time.Second*5),
	)
	assert.NoError(t, err)
	t.Log(err, iRegistry, serviceInstance)
}

func newConsulRegistry() *Registry {
	consulClient, err := api.NewClient(&api.Config{})
	if err != nil {
		panic(err)
	}

	r := New(consulClient, WithHealthCheck(true))

	return r
}

func TestRegistry_Register(t *testing.T) {
	r := newConsulRegistry()
	instance := registry.NewServiceInstance("foo", "bar", []string{"grpc://127.0.0.1:8282"})

	err := r.Register(context.Background(), instance)
	t.Log(err)

	_, err = r.ListServices()
	t.Log(err)

	_, err = r.GetService(context.Background(), "foo")
	t.Log(err)

	_, err = r.Watch(context.Background(), "foo")
	t.Log(err)

	go func() {
		r.resolve(newServiceSet())
	}()

	err = r.Deregister(context.Background(), instance)
	t.Log(err)

	time.Sleep(time.Millisecond * 100)
}
