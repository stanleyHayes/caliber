package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// ApplicationRepo is a Postgres-backed candidateagent.ApplicationRepository.
type ApplicationRepo struct {
	q *sqlcdb.Queries
}

// NewApplicationRepo builds the repository from a sqlc DBTX.
func NewApplicationRepo(db sqlcdb.DBTX) *ApplicationRepo { return &ApplicationRepo{q: sqlcdb.New(db)} }

// Create inserts a new application.
func (r *ApplicationRepo) Create(ctx context.Context, app *candidateagent.Application) error {
	err := r.q.CreateApplication(ctx, sqlcdb.CreateApplicationParams{
		ID:              app.ID.String(),
		RoleID:          app.RoleID.String(),
		CandidateID:     app.CandidateID.String(),
		ProfileID:       textOrNull(app.ProfileID.String(), !app.ProfileID.IsZero()),
		Source:          appSourceToDB(app.Source),
		TailoredSummary: textOrNull(app.TailoredSummary, app.TailoredSummary != ""),
		Status:          appStatusToDB(app.Status),
	})
	if isUniqueViolation(err) {
		return kernel.Conflict("postgres: application already exists")
	}
	return err
}

// ByID returns an application by id.
func (r *ApplicationRepo) ByID(ctx context.Context, id kernel.ID) (*candidateagent.Application, error) {
	row, err := r.q.GetApplication(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: application not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomainApplication(
		row.ID, row.RoleID, row.CandidateID, row.ProfileID,
		row.Source, row.TailoredSummary.String, row.Status,
	), nil
}

// Update persists changes to an existing application.
func (r *ApplicationRepo) Update(ctx context.Context, app *candidateagent.Application) error {
	n, err := r.q.UpdateApplication(ctx, sqlcdb.UpdateApplicationParams{
		ID:              app.ID.String(),
		Status:          appStatusToDB(app.Status),
		TailoredSummary: textOrNull(app.TailoredSummary, app.TailoredSummary != ""),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return kernel.NotFound("postgres: application not found")
	}
	return nil
}

// ByCandidate returns a page of a candidate's applications and the total count.
func (r *ApplicationRepo) ByCandidate(
	ctx context.Context, candidateID kernel.ID, page kernel.Page,
) ([]*candidateagent.Application, int64, error) {
	rows, err := r.q.ListApplicationsByCandidate(ctx, sqlcdb.ListApplicationsByCandidateParams{
		CandidateID: candidateID.String(),
		Limit:       clampInt32(page.Limit()),
		Offset:      clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*candidateagent.Application, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainApplication(
			row.ID, row.RoleID, row.CandidateID, row.ProfileID,
			row.Source, row.TailoredSummary.String, row.Status,
		))
	}
	total, err := r.q.CountApplicationsByCandidate(ctx, candidateID.String())
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func toDomainApplication(
	id, roleID, candidateID string, profileID pgtype.Text, source, tailored, status string,
) *candidateagent.Application {
	var pid kernel.ID
	if profileID.Valid {
		pid = kernel.ID(profileID.String)
	}
	return &candidateagent.Application{
		ID:              kernel.ID(id),
		RoleID:          kernel.ID(roleID),
		CandidateID:     kernel.ID(candidateID),
		ProfileID:       pid,
		Source:          appSourceFromDB(source),
		TailoredSummary: tailored,
		Status:          appStatusFromDB(status),
	}
}

func textOrNull(s string, valid bool) pgtype.Text { return pgtype.Text{String: s, Valid: valid} }

func appSourceToDB(s candidateagent.ApplicationSource) string {
	if s == candidateagent.SourceAgent {
		return "agent"
	}
	return "manual"
}

func appSourceFromDB(s string) candidateagent.ApplicationSource {
	if s == "agent" {
		return candidateagent.SourceAgent
	}
	return candidateagent.SourceManual
}

func appStatusToDB(s candidateagent.ApplicationStatus) string {
	switch s {
	case candidateagent.StatusSubmitted:
		return "submitted"
	case candidateagent.StatusScreening:
		return "screening"
	case candidateagent.StatusScreened:
		return "screened"
	default:
		return "drafted"
	}
}

func appStatusFromDB(s string) candidateagent.ApplicationStatus {
	switch s {
	case "submitted":
		return candidateagent.StatusSubmitted
	case "screening":
		return candidateagent.StatusScreening
	case "screened":
		return candidateagent.StatusScreened
	default:
		return candidateagent.StatusDrafted
	}
}

var _ candidateagent.ApplicationRepository = (*ApplicationRepo)(nil)
