// Package matching holds Matching application use-cases (Flow A shortlist).
package matching

import (
	"context"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

//go:generate mockgen -source=recaller.go -destination=../../mocks/recaller.go -package=mocks

// CandidateRecaller returns candidate ids whose profile embeddings are nearest
// to a role embedding (matching stage 1: vector recall). A pgvector adapter
// implements it in production; an in-memory recaller serves development.
type CandidateRecaller interface {
	Recall(ctx context.Context, roleEmbedding []float32, limit int) ([]kernel.ID, error)
}
