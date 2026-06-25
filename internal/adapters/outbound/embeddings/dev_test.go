package embeddings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevEmbedDeterministic(t *testing.T) {
	d := NewDev()
	a, err := d.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Len(t, a, 1536)

	b, err := d.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, a, b, "same text -> same vector")

	c, err := d.Embed(context.Background(), "world")
	require.NoError(t, err)
	assert.NotEqual(t, a, c, "different text -> different vector")
}
