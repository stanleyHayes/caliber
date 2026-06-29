package grpcadapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGRPCServer(t *testing.T) {
	require.NotNil(t, NewGRPCServer(Services{}))
	require.NotNil(t, NewGRPCServer(Services{Role: NewRoleServer(nil, nil, nil)}))
}

func TestReflectionIsGated(t *testing.T) {
	const reflectionSvc = "grpc.reflection.v1.ServerReflection"

	// Off (the prod default): the reflection service is not registered, so the API
	// schema is not advertised to anyone reaching the port (CAL-120 L4).
	off := NewGRPCServer(Services{EnableReflection: false})
	_, present := off.GetServiceInfo()[reflectionSvc]
	assert.False(t, present, "reflection must be absent when disabled")

	// On (dev/staging): registered for grpcurl/evans convenience.
	on := NewGRPCServer(Services{EnableReflection: true})
	_, present = on.GetServiceInfo()[reflectionSvc]
	assert.True(t, present, "reflection is available when enabled")
}

func TestDialTarget(t *testing.T) {
	assert.Equal(t, "localhost:9090", DialTarget(":9090"))
	assert.Equal(t, "host:9090", DialTarget("host:9090"))
}
