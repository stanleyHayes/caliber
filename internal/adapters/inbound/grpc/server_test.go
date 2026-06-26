package grpcadapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGRPCServer(t *testing.T) {
	require.NotNil(t, NewGRPCServer(Services{}))
	require.NotNil(t, NewGRPCServer(Services{Role: NewRoleServer(nil, nil)}))
}

func TestDialTarget(t *testing.T) {
	assert.Equal(t, "localhost:9090", DialTarget(":9090"))
	assert.Equal(t, "host:9090", DialTarget("host:9090"))
}
