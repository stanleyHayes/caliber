package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/matching"
)

// MatchRepo is a Postgres-backed matching.MatchRepository.
type MatchRepo struct {
	q *sqlcdb.Queries
}

// NewMatchRepo builds the repository from a sqlc DBTX.
func NewMatchRepo(db sqlcdb.DBTX) *MatchRepo { return &MatchRepo{q: sqlcdb.New(db)} }

// Upsert inserts or updates a match (keyed by role + candidate).
func (r *MatchRepo) Upsert(ctx context.Context, m *matching.Match) error {
	breakdown, err := json.Marshal(m.Breakdown)
	if err != nil {
		return err
	}
	watchOuts, err := json.Marshal(m.WatchOuts)
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
		Rationale:        pgtype.Text{String: m.Rationale, Valid: true},
		WatchOuts:        watchOuts,
		ThinEvidenceFlag: m.ThinEvidence,
	})
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
	out, err := toDomainMatches(rows)
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
	out, err := toDomainMatches(rows)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountMatchesByCandidate(ctx, candidateID.String())
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func toDomainMatches(rows []sqlcdb.Match) ([]*matching.Match, error) {
	out := make([]*matching.Match, 0, len(rows))
	for _, row := range rows {
		m, err := toDomainMatch(row)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func toDomainMatch(row sqlcdb.Match) (*matching.Match, error) {
	var breakdown []matching.MatchBreakdownItem
	if len(row.Breakdown) > 0 {
		if err := json.Unmarshal(row.Breakdown, &breakdown); err != nil {
			return nil, err
		}
	}
	var watchOuts []string
	if len(row.WatchOuts) > 0 {
		if err := json.Unmarshal(row.WatchOuts, &watchOuts); err != nil {
			return nil, err
		}
	}
	return &matching.Match{
		ID:           kernel.ID(row.ID),
		RoleID:       kernel.ID(row.RoleID),
		CandidateID:  kernel.ID(row.CandidateID),
		OverallScore: row.OverallScore,
		Confidence:   confidenceFromDB(row.Confidence),
		Breakdown:    breakdown,
		Rationale:    row.Rationale.String,
		WatchOuts:    watchOuts,
		ThinEvidence: row.ThinEvidenceFlag,
	}, nil
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

var _ matching.MatchRepository = (*MatchRepo)(nil)
