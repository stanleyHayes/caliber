package matching

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/xcreativs/caliber/internal/app"
)

// CachedEmbedder wraps an Embedder with a small in-process cache keyed by the
// hash of the input text. The shortlist path embeds the same role text twice
// (GenerateShortlist + CountAvailable) and may re-rank after edits, so caching
// the role embedding avoids redundant provider calls and keeps the path fast
// (CAL-104).
type CachedEmbedder struct {
	inner app.Embedder
	mu    sync.Mutex
	cache map[string][]float32
}

// NewCachedEmbedder wraps inner with a hash-keyed embedding cache.
func NewCachedEmbedder(inner app.Embedder) *CachedEmbedder {
	return &CachedEmbedder{inner: inner, cache: make(map[string][]float32)}
}

// Embed returns a cached vector when the same text was embedded recently;
// otherwise it delegates to the inner embedder and stores the result.
func (c *CachedEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	key := hashText(text)
	c.mu.Lock()
	if v, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()

	v, err := c.inner.Embed(ctx, text)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = v
	c.mu.Unlock()
	return v, nil
}

func hashText(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

var _ app.Embedder = (*CachedEmbedder)(nil)
