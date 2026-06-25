package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// CandidateRepo is a Postgres-backed talent.CandidateRepository.
type CandidateRepo struct {
	q *sqlcdb.Queries
}

// NewCandidateRepo builds the repository from a sqlc DBTX.
func NewCandidateRepo(db sqlcdb.DBTX) *CandidateRepo { return &CandidateRepo{q: sqlcdb.New(db)} }

// Create inserts a new candidate.
func (r *CandidateRepo) Create(ctx context.Context, c *talent.Candidate) error {
	prefs, err := json.Marshal(c.Intake)
	if err != nil {
		return err
	}
	err = r.q.CreateCandidate(ctx, sqlcdb.CreateCandidateParams{
		ID:          c.ID.String(),
		UserID:      c.UserID.String(),
		Location:    pgtype.Text{String: c.Location, Valid: true},
		Preferences: prefs,
	})
	if isUniqueViolation(err) {
		return kernel.Conflict("postgres: candidate already exists")
	}
	return err
}

// ByID returns a candidate by id.
func (r *CandidateRepo) ByID(ctx context.Context, id kernel.ID) (*talent.Candidate, error) {
	row, err := r.q.GetCandidate(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: candidate not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomainCandidate(row.ID, row.UserID, row.Location, row.Preferences)
}

// ByUserID returns the candidate belonging to a user.
func (r *CandidateRepo) ByUserID(ctx context.Context, userID kernel.ID) (*talent.Candidate, error) {
	row, err := r.q.GetCandidateByUserID(ctx, userID.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: candidate not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomainCandidate(row.ID, row.UserID, row.Location, row.Preferences)
}

// Update persists changes to an existing candidate.
func (r *CandidateRepo) Update(ctx context.Context, c *talent.Candidate) error {
	prefs, err := json.Marshal(c.Intake)
	if err != nil {
		return err
	}
	n, err := r.q.UpdateCandidate(ctx, sqlcdb.UpdateCandidateParams{
		ID:          c.ID.String(),
		Location:    pgtype.Text{String: c.Location, Valid: true},
		Preferences: prefs,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return kernel.NotFound("postgres: candidate not found")
	}
	return nil
}

// List returns a page of candidates and the total count.
func (r *CandidateRepo) List(ctx context.Context, page kernel.Page) ([]*talent.Candidate, int64, error) {
	rows, err := r.q.ListCandidates(ctx, sqlcdb.ListCandidatesParams{
		Limit:  clampInt32(page.Limit()),
		Offset: clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*talent.Candidate, 0, len(rows))
	for _, row := range rows {
		c, derr := toDomainCandidate(row.ID, row.UserID, row.Location, row.Preferences)
		if derr != nil {
			return nil, 0, derr
		}
		out = append(out, c)
	}
	total, err := r.q.CountCandidates(ctx)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func toDomainCandidate(id, userID string, location pgtype.Text, prefs []byte) (*talent.Candidate, error) {
	var intake talent.CandidateIntake
	if len(prefs) > 0 {
		if err := json.Unmarshal(prefs, &intake); err != nil {
			return nil, err
		}
	}
	return &talent.Candidate{
		ID:       kernel.ID(id),
		UserID:   kernel.ID(userID),
		Location: location.String,
		Intake:   intake,
	}, nil
}

var _ talent.CandidateRepository = (*CandidateRepo)(nil)
