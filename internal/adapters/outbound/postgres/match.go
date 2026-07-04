package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/matching"
)

// MatchRepo is a Postgres-backed matching.MatchRepository. The match rationale,
// watch-outs, and breakdown evidence are candidate-derived assessment prose
// (restating skills, gaps, and inferred attributes), so they are encrypted at
// rest when a field cipher is configured (CAL-117).
type MatchRepo struct {
	q      *sqlcdb.Queries
	cipher *fieldcrypto.FieldCipher
}

// MatchOption customizes a MatchRepo.
type MatchOption func(*MatchRepo)

// WithMatchCipher encrypts the match rationale/watch-outs/breakdown at rest with
// the given cipher (CAL-117). A nil cipher is ignored (passthrough default kept).
func WithMatchCipher(c *fieldcrypto.FieldCipher) MatchOption {
	return func(r *MatchRepo) {
		if c != nil {
			r.cipher = c
		}
	}
}

// NewMatchRepo builds the repository from a sqlc DBTX. Without a cipher option the
// assessment prose is stored as plaintext (passthrough).
func NewMatchRepo(db sqlcdb.DBTX, opts ...MatchOption) *MatchRepo {
	r := &MatchRepo{q: sqlcdb.New(db), cipher: fieldcrypto.Passthrough()}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Upsert inserts or updates a match (keyed by role + candidate).
func (r *MatchRepo) Upsert(ctx context.Context, m *matching.Match) error {
	breakdown, err := encodeEncryptedJSON(r.cipher, m.Breakdown)
	if err != nil {
		return err
	}
	watchOuts, err := encodeEncryptedJSON(r.cipher, m.WatchOuts)
	if err != nil {
		return err
	}
	rationale, err := r.cipher.Encrypt(m.Rationale)
	if err != nil {
		return err
	}
	return r.q.UpsertMatch(ctx, sqlcdb.UpsertMatchParams{
		ID:               m.ID.String(),
		RoleID:           m.RoleID.String(),
		CandidateID:      m.CandidateID.String(),
		OverallScore:     m.OverallScore,
		Confidence:       confidenceToDB(m.Confidence),
		Breakdown:        breakdown,
		Rationale:        pgtype.Text{String: rationale, Valid: true},
		WatchOuts:        watchOuts,
		ThinEvidenceFlag: m.ThinEvidence,
	})
}

// ByID returns a single match by id, or kernel.KindNotFound when absent.
func (r *MatchRepo) ByID(ctx context.Context, id kernel.ID) (*matching.Match, error) {
	row, err := r.q.MatchByID(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: match not found")
	}
	if err != nil {
		return nil, err
	}
	return r.toDomainMatch(row)
}

// ByRole returns a page of matches for a role and the total count.
func (r *MatchRepo) ByRole(ctx context.Context, roleID kernel.ID, page kernel.Page) ([]*matching.Match, int64, error) {
	rows, err := r.q.ListMatchesByRole(ctx, sqlcdb.ListMatchesByRoleParams{
		RoleID: roleID.String(),
		Limit:  clampInt32(page.Limit()),
		Offset: clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out, err := r.toDomainMatches(rows)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountMatchesByRole(ctx, roleID.String())
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ForCandidate returns a page of matches for a candidate and the total count.
func (r *MatchRepo) ForCandidate(ctx context.Context, candidateID kernel.ID, page kernel.Page) ([]*matching.Match, int64, error) {
	rows, err := r.q.ListMatchesByCandidate(ctx, sqlcdb.ListMatchesByCandidateParams{
		CandidateID: candidateID.String(),
		Limit:       clampInt32(page.Limit()),
		Offset:      clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out, err := r.toDomainMatches(rows)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountMatchesByCandidate(ctx, candidateID.String())
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func confidenceToDB(c kernel.Confidence) string {
	switch c {
	case kernel.ConfidenceLow:
		return "low"
	case kernel.ConfidenceMedium:
		return "medium"
	case kernel.ConfidenceHigh:
		return "high"
	default:
		return ""
	}
}

func confidenceFromDB(s string) kernel.Confidence {
	switch s {
	case "low":
		return kernel.ConfidenceLow
	case "medium":
		return kernel.ConfidenceMedium
	case "high":
		return kernel.ConfidenceHigh
	default:
		return kernel.ConfidenceUnknown
	}
}

// DeleteByCandidate hard-deletes every match referencing a candidate
// (CAL-118 right-to-erasure).
func (r *MatchRepo) DeleteByCandidate(ctx context.Context, candidateID kernel.ID) error {
	return r.q.DeleteMatchesByCandidate(ctx, candidateID.String())
}

func (r *MatchRepo) toDomainMatches(rows []sqlcdb.Match) ([]*matching.Match, error) {
	out := make([]*matching.Match, 0, len(rows))
	for _, row := range rows {
		m, err := r.toDomainMatch(row)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *MatchRepo) toDomainMatch(row sqlcdb.Match) (*matching.Match, error) {
	var breakdown []matching.MatchBreakdownItem
	if err := decodeEncryptedJSON(r.cipher, row.Breakdown, &breakdown); err != nil {
		return nil, err
	}
	var watchOuts []string
	if err := decodeEncryptedJSON(r.cipher, row.WatchOuts, &watchOuts); err != nil {
		return nil, err
	}
	rationale, err := r.cipher.Decrypt(row.Rationale.String)
	if err != nil {
		return nil, err
	}
	return &matching.Match{
		ID:           kernel.ID(row.ID),
		RoleID:       kernel.ID(row.RoleID),
		CandidateID:  kernel.ID(row.CandidateID),
		OverallScore: row.OverallScore,
		Confidence:   confidenceFromDB(row.Confidence),
		Breakdown:    breakdown,
		Rationale:    rationale,
		WatchOuts:    watchOuts,
		ThinEvidence: row.ThinEvidenceFlag,
		CreatedAt:    row.CreatedAt.Time,
	}, nil
}

var _ matching.MatchRepository = (*MatchRepo)(nil)
