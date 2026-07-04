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

// TalentProfileRepo is a Postgres-backed talent.TalentProfileRepository. The
// free-text summary is candidate PII, so it is encrypted at rest when a field
// cipher is configured (CAL-117).
type TalentProfileRepo struct {
	q      *sqlcdb.Queries
	cipher *fieldcrypto.FieldCipher
}

// TalentProfileOption customizes a TalentProfileRepo.
type TalentProfileOption func(*TalentProfileRepo)

// WithFieldCipher encrypts the profile summary at rest with the given cipher
// (CAL-117). A nil cipher is ignored (the passthrough default is kept).
func WithFieldCipher(c *fieldcrypto.FieldCipher) TalentProfileOption {
	return func(r *TalentProfileRepo) {
		if c != nil {
			r.cipher = c
		}
	}
}

// NewTalentProfileRepo builds the repository from a sqlc DBTX. Without a cipher
// option the summary is stored as plaintext (passthrough).
func NewTalentProfileRepo(db sqlcdb.DBTX, opts ...TalentProfileOption) *TalentProfileRepo {
	r := &TalentProfileRepo{q: sqlcdb.New(db), cipher: fieldcrypto.Passthrough()}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Create inserts a new talent profile.
func (r *TalentProfileRepo) Create(ctx context.Context, p *talent.TalentProfile) error {
	comps, err := json.Marshal(p.Competencies)
	if err != nil {
		return err
	}
	summary, err := r.cipher.Encrypt(p.Summary)
	if err != nil {
		return err
	}
	err = r.q.CreateTalentProfile(ctx, sqlcdb.CreateTalentProfileParams{
		ID:             p.ID.String(),
		CandidateID:    p.CandidateID.String(),
		Summary:        pgtype.Text{String: summary, Valid: true},
		Profile:        comps,
		Column5:        vectorLiteralOrEmpty(p.Embedding),
		PassportStatus: passportToDB(p.PassportStatus),
	})
	if isUniqueViolation(err) {
		return kernel.Conflict("postgres: talent profile already exists")
	}
	return err
}

// ByID returns a talent profile by id.
func (r *TalentProfileRepo) ByID(ctx context.Context, id kernel.ID) (*talent.TalentProfile, error) {
	row, err := r.q.GetTalentProfile(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: talent profile not found")
	}
	if err != nil {
		return nil, err
	}
	return r.toDomainTalentProfile(row.ID, row.CandidateID, row.Summary, row.Profile, row.PassportStatus)
}

// ByCandidateID returns the talent profile for a candidate.
func (r *TalentProfileRepo) ByCandidateID(ctx context.Context, candidateID kernel.ID) (*talent.TalentProfile, error) {
	row, err := r.q.GetTalentProfileByCandidateID(ctx, candidateID.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: talent profile not found")
	}
	if err != nil {
		return nil, err
	}
	return r.toDomainTalentProfile(row.ID, row.CandidateID, row.Summary, row.Profile, row.PassportStatus)
}

// Update persists changes to an existing talent profile.
func (r *TalentProfileRepo) Update(ctx context.Context, p *talent.TalentProfile) error {
	comps, err := json.Marshal(p.Competencies)
	if err != nil {
		return err
	}
	summary, err := r.cipher.Encrypt(p.Summary)
	if err != nil {
		return err
	}
	n, err := r.q.UpdateTalentProfile(ctx, sqlcdb.UpdateTalentProfileParams{
		ID:             p.ID.String(),
		Summary:        pgtype.Text{String: summary, Valid: true},
		Profile:        comps,
		Column4:        vectorLiteralOrEmpty(p.Embedding),
		PassportStatus: passportToDB(p.PassportStatus),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return kernel.NotFound("postgres: talent profile not found")
	}
	return nil
}

// List returns a page of talent profiles and the total count.
func (r *TalentProfileRepo) List(ctx context.Context, page kernel.Page) ([]*talent.TalentProfile, int64, error) {
	rows, err := r.q.ListTalentProfiles(ctx, sqlcdb.ListTalentProfilesParams{
		Limit:  clampInt32(page.Limit()),
		Offset: clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*talent.TalentProfile, 0, len(rows))
	for _, row := range rows {
		p, derr := r.toDomainTalentProfile(row.ID, row.CandidateID, row.Summary, row.Profile, row.PassportStatus)
		if derr != nil {
			return nil, 0, derr
		}
		out = append(out, p)
	}
	total, err := r.q.CountTalentProfiles(ctx)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func passportToDB(s talent.PassportStatus) string {
	switch s {
	case talent.PassportScreened:
		return "screened"
	case talent.PassportVerified:
		return "verified"
	default:
		return "cv_only"
	}
}

func passportFromDB(s string) talent.PassportStatus {
	switch s {
	case "screened":
		return talent.PassportScreened
	case "verified":
		return talent.PassportVerified
	default:
		return talent.PassportCVOnly
	}
}

// DeleteByCandidate hard-deletes a candidate's talent profile and its embedding
// (CAL-118 right-to-erasure).
func (r *TalentProfileRepo) DeleteByCandidate(ctx context.Context, candidateID kernel.ID) error {
	return r.q.DeleteTalentProfilesByCandidate(ctx, candidateID.String())
}

func (r *TalentProfileRepo) toDomainTalentProfile(
	id, candidateID string, summary pgtype.Text, profile []byte, passport string,
) (*talent.TalentProfile, error) {
	var comps []talent.ProfileCompetency
	if len(profile) > 0 {
		if err := json.Unmarshal(profile, &comps); err != nil {
			return nil, err
		}
	}
	plainSummary, err := r.cipher.Decrypt(summary.String)
	if err != nil {
		return nil, err
	}
	return &talent.TalentProfile{
		ID:             kernel.ID(id),
		CandidateID:    kernel.ID(candidateID),
		Summary:        plainSummary,
		Competencies:   comps,
		PassportStatus: passportFromDB(passport),
	}, nil
}

var _ talent.TalentProfileRepository = (*TalentProfileRepo)(nil)
