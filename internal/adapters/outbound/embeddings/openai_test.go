package embeddings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer srv.Close()

	e := NewOpenAI(WithOpenAIKey("test"), WithOpenAIBaseURL(srv.URL), WithOpenAIModel("text-embedding-3-small"))
	v, err := e.Embed(context.Background(), "hello")
	require.NoError(t, err)
	require.Len(t, v, 3)
	assert.InDelta(t, 0.2, v[1], 1e-6)
}

func TestOpenAIEmbedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()
	_, err := NewOpenAI(WithOpenAIKey("x"), WithOpenAIBaseURL(srv.URL)).Embed(context.Background(), "x")
	require.Error(t, err)
}

func TestOpenAIEmbedEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	_, err := NewOpenAI(WithOpenAIKey("x"), WithOpenAIBaseURL(srv.URL)).Embed(context.Background(), "x")
	require.Error(t, err)
}
