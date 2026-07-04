package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// CandidateRepo is a Postgres-backed talent.CandidateRepository. A candidate's
// location and intake preferences (salary floor, deal-breakers, target titles)
// are PII, so both are encrypted at rest when a field cipher is configured
// (CAL-117), mirroring the talent-profile summary.
type CandidateRepo struct {
	q      *sqlcdb.Queries
	cipher *fieldcrypto.FieldCipher
}

// CandidateOption customizes a CandidateRepo.
type CandidateOption func(*CandidateRepo)

// WithCandidateCipher encrypts candidate location and intake preferences at rest
// with the given cipher (CAL-117). A nil cipher is ignored (passthrough kept).
func WithCandidateCipher(c *fieldcrypto.FieldCipher) CandidateOption {
	return func(r *CandidateRepo) {
		if c != nil {
			r.cipher = c
		}
	}
}

// NewCandidateRepo builds the repository from a sqlc DBTX. Without a cipher
// option location and preferences are stored as plaintext (passthrough).
func NewCandidateRepo(db sqlcdb.DBTX, opts ...CandidateOption) *CandidateRepo {
	r := &CandidateRepo{q: sqlcdb.New(db), cipher: fieldcrypto.Passthrough()}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Create inserts a new candidate.
func (r *CandidateRepo) Create(ctx context.Context, c *talent.Candidate) error {
	prefs, err := r.encodeIntake(c.Intake)
	if err != nil {
		return err
	}
	location, err := r.cipher.Encrypt(c.Location)
	if err != nil {
		return err
	}
	err = r.q.CreateCandidate(ctx, sqlcdb.CreateCandidateParams{
		ID:          c.ID.String(),
		UserID:      c.UserID.String(),
		Location:    pgtype.Text{String: location, Valid: true},
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
	return r.toDomainCandidate(row.ID, row.UserID, row.Location, row.Preferences)
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
	return r.toDomainCandidate(row.ID, row.UserID, row.Location, row.Preferences)
}

// Update persists changes to an existing candidate.
func (r *CandidateRepo) Update(ctx context.Context, c *talent.Candidate) error {
	prefs, err := r.encodeIntake(c.Intake)
	if err != nil {
		return err
	}
	location, err := r.cipher.Encrypt(c.Location)
	if err != nil {
		return err
	}
	n, err := r.q.UpdateCandidate(ctx, sqlcdb.UpdateCandidateParams{
		ID:          c.ID.String(),
		Location:    pgtype.Text{String: location, Valid: true},
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
		c, derr := r.toDomainCandidate(row.ID, row.UserID, row.Location, row.Preferences)
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

// Delete hard-deletes a candidate (CAL-118 right-to-erasure). The FK
// ON DELETE CASCADE removes dependent profiles, matches, interviews, and
// applications; the erasure use-case also deletes those explicitly first.
func (r *CandidateRepo) Delete(ctx context.Context, id kernel.ID) error {
	return r.q.DeleteCandidate(ctx, id.String())
}

func (r *CandidateRepo) toDomainCandidate(id, userID string, location pgtype.Text, prefs []byte) (*talent.Candidate, error) {
	intake, err := r.decodeIntake(prefs)
	if err != nil {
		return nil, err
	}
	plainLocation, err := r.cipher.Decrypt(location.String)
	if err != nil {
		return nil, err
	}
	return &talent.Candidate{
		ID:       kernel.ID(id),
		UserID:   kernel.ID(userID),
		Location: plainLocation,
		Intake:   intake,
	}, nil
}

// encodeIntake marshals the intake for the JSONB preferences column. In
// passthrough mode the object is stored as-is (byte-identical to legacy rows).
// With a key configured the ciphertext is wrapped as a JSON string so the
// column stays valid JSON while the PII (salary floor, deal-breakers) is opaque.
func (r *CandidateRepo) encodeIntake(intake talent.CandidateIntake) ([]byte, error) {
	raw, err := json.Marshal(intake)
	if err != nil {
		return nil, err
	}
	if !r.cipher.Enabled() {
		return raw, nil
	}
	enc, err := r.cipher.Encrypt(string(raw))
	if err != nil {
		return nil, err
	}
	return json.Marshal(enc)
}

// decodeIntake reverses encodeIntake. A JSONB string value is a (possibly
// encrypted) wrapped payload; a JSONB object is the legacy/passthrough form
// stored directly. CandidateIntake never marshals to a bare JSON string, so the
// two are unambiguous.
func (r *CandidateRepo) decodeIntake(prefs []byte) (talent.CandidateIntake, error) {
	var intake talent.CandidateIntake
	if len(prefs) == 0 {
		return intake, nil
	}
	var wrapped string
	if err := json.Unmarshal(prefs, &wrapped); err == nil {
		plain, derr := r.cipher.Decrypt(wrapped)
		if derr != nil {
			return intake, derr
		}
		return intake, json.Unmarshal([]byte(plain), &intake)
	}
	return intake, json.Unmarshal(prefs, &intake)
}

var _ talent.CandidateRepository = (*CandidateRepo)(nil)
