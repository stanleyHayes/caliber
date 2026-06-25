package postgres

import (
	"context"
	"strconv"
	"strings"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

const recallSQL = `SELECT candidate_id FROM talent_profiles
WHERE profile_embedding IS NOT NULL
ORDER BY profile_embedding <=> $1::vector
LIMIT $2`

// Recaller is a pgvector-backed matchingapp.CandidateRecaller (matching stage 1:
// cosine-distance recall of candidate profiles nearest a role embedding).
type Recaller struct {
	db sqlcdb.DBTX
}

// NewRecaller builds the recaller from a sqlc DBTX (e.g. a *pgxpool.Pool).
func NewRecaller(db sqlcdb.DBTX) *Recaller { return &Recaller{db: db} }

// Recall returns candidate ids whose profile embeddings are nearest the role embedding.
func (r *Recaller) Recall(ctx context.Context, roleEmbedding []float32, limit int) ([]kernel.ID, error) {
	rows, err := r.db.Query(ctx, recallSQL, vectorLiteral(roleEmbedding), clampInt32(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]kernel.ID, 0, max(limit, 0))
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, kernel.ID(id))
	}
	return out, rows.Err()
}

// vectorLiteral renders a float slice as a pgvector text literal: [0.1,0.2,...].
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

var _ matchingapp.CandidateRecaller = (*Recaller)(nil)
