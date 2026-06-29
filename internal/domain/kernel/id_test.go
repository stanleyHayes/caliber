package kernel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDFromString(t *testing.T) {
	id, err := IDFromString("candidate-123")
	require.NoError(t, err)
	assert.Equal(t, ID("candidate-123"), id)
}

func TestIDFromStringEmpty(t *testing.T) {
	_, err := IDFromString("")
	require.Error(t, err)
	_, err = IDFromString("   ")
	require.Error(t, err)
}

func TestIDStringAndIsZero(t *testing.T) {
	id := ID("x")
	assert.Equal(t, "x", id.String())
	assert.False(t, id.IsZero())
	assert.True(t, ID("").IsZero())
	assert.True(t, ID("  ").IsZero())
}
