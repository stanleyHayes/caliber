package embeddings

import (
	"context"
	"crypto/sha256"

	"github.com/xcreativs/caliber/internal/app"
)

// devDim matches OpenAI text-embedding-3-small (1536).
const devDim = 1536

// Dev is a deterministic, offline app.Embedder for local development and tests.
type Dev struct {
	dim int
}

// NewDev returns a deterministic dev embedder (1536 dimensions).
func NewDev() *Dev { return &Dev{dim: devDim} }

// Embed returns a deterministic pseudo-embedding derived from the text.
func (d *Dev) Embed(_ context.Context, text string) ([]float32, error) {
	sum := sha256.Sum256([]byte(text))
	out := make([]float32, d.dim)
	for i := range out {
		out[i] = float32(sum[i%len(sum)]) / 255.0
	}
	return out, nil
}

var _ app.Embedder = (*Dev)(nil)
