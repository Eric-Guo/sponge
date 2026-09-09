package tracer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestNewResource(t *testing.T) {
	res := NewResource(WithServiceName("rails-proxy"), WithServiceVersion("v1"))
	assert.NotNil(t, res)
	assert.Equal(t, resource.Default().SchemaURL(), res.SchemaURL())
	name, ok := res.Set().Value(attribute.Key("service.name"))
	assert.True(t, ok)
	assert.Equal(t, "rails-proxy", name.AsString())
}

func TestWithAttributes(t *testing.T) {
	testData := map[string]string{}
	o := new(resourceOptions)
	opt := WithAttributes(testData)
	apply(o, opt)
	assert.Equal(t, testData, o.attributes)
}

func TestWithEnvironment(t *testing.T) {
	testData := "env"
	o := new(resourceOptions)
	opt := WithEnvironment(testData)
	apply(o, opt)
	assert.Equal(t, testData, o.environment)
}

func TestWithServiceName(t *testing.T) {
	testData := "foo"
	o := new(resourceOptions)
	opt := WithServiceName(testData)
	apply(o, opt)
	assert.Equal(t, testData, o.serviceName)
}

func TestWithServiceVersion(t *testing.T) {
	testData := "v1.0"
	o := new(resourceOptions)
	opt := WithServiceVersion(testData)
	apply(o, opt)
	assert.Equal(t, testData, o.serviceVersion)
}

func Test_apply(t *testing.T) {
	testData := "v1.0"
	o := new(resourceOptions)
	opt := WithServiceVersion(testData)
	apply(o, opt)
	assert.Equal(t, testData, o.serviceVersion)
}

func Test_resourceOptionFunc_apply(t *testing.T) {
	testData := "v1.0"
	o := new(resourceOptions)
	opt := WithServiceVersion(testData)
	apply(o, opt)
	assert.Equal(t, testData, o.serviceVersion)
}
