// Package postgres provides pgx + sqlc repository adapters implementing the
// domain repository ports. LLM-produced structures are stored as JSONB.
package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

const uniqueViolation = "23505"

// RoleRepo is a Postgres-backed role.RoleRepository.
type RoleRepo struct {
	q *sqlcdb.Queries
}

// NewRoleRepo builds the repository from a sqlc DBTX (e.g. a *pgxpool.Pool).
func NewRoleRepo(db sqlcdb.DBTX) *RoleRepo {
	return &RoleRepo{q: sqlcdb.New(db)}
}

// Create inserts a new role.
func (r *RoleRepo) Create(ctx context.Context, rl *role.Role) error {
	spec, rubric, band, err := marshalRole(rl)
	if err != nil {
		return err
	}
	err = r.q.CreateRole(ctx, sqlcdb.CreateRoleParams{
		ID:         rl.ID.String(),
		EmployerID: rl.EmployerID.String(),
		Title:      rl.Title,
		Status:     statusToDB(rl.Status),
		RoleSpec:   spec,
		Rubric:     rubric,
		SalaryBand: band,
		CreatedAt:  pgtype.Timestamptz{Time: rl.CreatedAt, Valid: true},
	})
	if isUniqueViolation(err) {
		return kernel.Conflict("postgres: role already exists")
	}
	return err
}

// ByID returns a role by id.
func (r *RoleRepo) ByID(ctx context.Context, id kernel.ID) (*role.Role, error) {
	row, err := r.q.GetRole(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: role not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomainRole(row.ID, row.EmployerID, row.Title, row.Status, row.RoleSpec, row.Rubric, row.CreatedAt)
}

// Update replaces an existing role; returns NotFound if it does not exist.
func (r *RoleRepo) Update(ctx context.Context, rl *role.Role) error {
	spec, rubric, band, err := marshalRole(rl)
	if err != nil {
		return err
	}
	n, err := r.q.UpdateRole(ctx, sqlcdb.UpdateRoleParams{
		ID:         rl.ID.String(),
		Title:      rl.Title,
		Status:     statusToDB(rl.Status),
		RoleSpec:   spec,
		Rubric:     rubric,
		SalaryBand: band,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return kernel.NotFound("postgres: role not found")
	}
	return nil
}

// ListOpen lists non-closed roles (the applyable pool), newest first.
func (r *RoleRepo) ListOpen(ctx context.Context, page kernel.Page) ([]*role.Role, int64, error) {
	rows, err := r.q.ListOpenRoles(ctx, sqlcdb.ListOpenRolesParams{
		Limit:  clampInt32(page.Limit()),
		Offset: clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*role.Role, 0, len(rows))
	for _, row := range rows {
		rl, derr := toDomainRole(row.ID, row.EmployerID, row.Title, row.Status, row.RoleSpec, row.Rubric, row.CreatedAt)
		if derr != nil {
			return nil, 0, derr
		}
		out = append(out, rl)
	}
	total, err := r.q.CountOpenRoles(ctx)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ListByEmployer returns a page of an employer's roles, newest first, plus the total.
func (r *RoleRepo) ListByEmployer(ctx context.Context, employerID kernel.ID, page kernel.Page) ([]*role.Role, int64, error) {
	rows, err := r.q.ListRolesByEmployer(ctx, sqlcdb.ListRolesByEmployerParams{
		EmployerID: employerID.String(),
		Limit:      clampInt32(page.Limit()),
		Offset:     clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*role.Role, 0, len(rows))
	for _, row := range rows {
		rl, derr := toDomainRole(row.ID, row.EmployerID, row.Title, row.Status, row.RoleSpec, row.Rubric, row.CreatedAt)
		if derr != nil {
			return nil, 0, derr
		}
		out = append(out, rl)
	}
	total, err := r.q.CountRolesByEmployer(ctx, employerID.String())
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func marshalRole(rl *role.Role) ([]byte, []byte, []byte, error) {
	spec, err := json.Marshal(rl.Spec)
	if err != nil {
		return nil, nil, nil, err
	}
	rubric, err := json.Marshal(rl.Rubric)
	if err != nil {
		return nil, nil, nil, err
	}
	band, err := json.Marshal(rl.Spec.SalaryBand)
	if err != nil {
		return nil, nil, nil, err
	}
	return spec, rubric, band, nil
}

func toDomainRole(id, employerID, title, status string, specJSON, rubricJSON []byte, createdAt pgtype.Timestamptz) (*role.Role, error) {
	var spec role.RoleSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, err
	}
	var rubric role.Rubric
	if err := json.Unmarshal(rubricJSON, &rubric); err != nil {
		return nil, err
	}
	return &role.Role{
		ID:         kernel.ID(id),
		EmployerID: kernel.ID(employerID),
		Title:      title,
		Status:     statusFromDB(status),
		Spec:       spec,
		Rubric:     rubric,
		CreatedAt:  createdAt.Time,
	}, nil
}

func statusToDB(s role.RoleStatus) string {
	switch s {
	case role.RoleOpen:
		return "open"
	case role.RoleClosed:
		return "closed"
	default:
		return "draft"
	}
}

func statusFromDB(s string) role.RoleStatus {
	switch s {
	case "open":
		return role.RoleOpen
	case "closed":
		return role.RoleClosed
	default:
		return role.RoleDraft
	}
}

// clampInt32 narrows a non-negative, already-bounded page value (kernel.Page
// clamps size to <= 100) to int32 for the generated query params.
func clampInt32(n int) int32 {
	if n < 0 {
		return 0
	}
	if n > 1<<31-1 {
		return 1<<31 - 1
	}
	return int32(n)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

var _ role.RoleRepository = (*RoleRepo)(nil)
