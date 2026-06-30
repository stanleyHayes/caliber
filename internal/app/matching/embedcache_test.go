package matching_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/mocks"
)

func TestCachedEmbedderHitsCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	inner := mocks.NewMockEmbedder(ctrl)
	cached := matchingapp.NewCachedEmbedder(inner)

	inner.EXPECT().Embed(gomock.Any(), "role text").Return([]float32{0.1, 0.2}, nil).Times(1)

	ctx := context.Background()
	first, err := cached.Embed(ctx, "role text")
	require.NoError(t, err)
	assert.Equal(t, []float32{0.1, 0.2}, first)

	second, err := cached.Embed(ctx, "role text")
	require.NoError(t, err)
	assert.Equal(t, first, second, "second call must be served from cache")
}

func TestCachedEmbedderKeysByText(t *testing.T) {
	ctrl := gomock.NewController(t)
	inner := mocks.NewMockEmbedder(ctrl)
	cached := matchingapp.NewCachedEmbedder(inner)

	gomock.InOrder(
		inner.EXPECT().Embed(gomock.Any(), "text a").Return([]float32{0.1}, nil),
		inner.EXPECT().Embed(gomock.Any(), "text b").Return([]float32{0.2}, nil),
	)

	ctx := context.Background()
	a, err := cached.Embed(ctx, "text a")
	require.NoError(t, err)
	assert.Equal(t, []float32{0.1}, a)

	b, err := cached.Embed(ctx, "text b")
	require.NoError(t, err)
	assert.Equal(t, []float32{0.2}, b)
}

func TestCachedEmbedderPropagatesErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	inner := mocks.NewMockEmbedder(ctrl)
	cached := matchingapp.NewCachedEmbedder(inner)

	inner.EXPECT().Embed(gomock.Any(), gomock.Any()).Return(nil, errors.New("embedder down")).Times(1)

	_, err := cached.Embed(context.Background(), "text")
	require.Error(t, err)

	// A failed call must not poison the cache: a retry should call the inner
	// embedder again.
	inner.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.9}, nil).Times(1)
	v, err := cached.Embed(context.Background(), "text")
	require.NoError(t, err)
	assert.Equal(t, []float32{0.9}, v)
}
