package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// UserRepo is a Postgres-backed identity.UserRepository.
type UserRepo struct {
	q *sqlcdb.Queries
}

// NewUserRepo builds the repository from a sqlc DBTX.
func NewUserRepo(db sqlcdb.DBTX) *UserRepo { return &UserRepo{q: sqlcdb.New(db)} }

// Create inserts a new user.
func (r *UserRepo) Create(ctx context.Context, u *identity.User) error {
	err := r.q.CreateUser(ctx, sqlcdb.CreateUserParams{
		ID:           u.ID.String(),
		Email:        u.Email.String(),
		Role:         userRoleToDB(u.Role),
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Status:       userStatusToDB(u.Status),
		CreatedAt:    pgtype.Timestamptz{Time: u.CreatedAt, Valid: true},
	})
	if isUniqueViolation(err) {
		return kernel.Conflict("postgres: user already exists")
	}
	return err
}

// ByID returns a user by id.
func (r *UserRepo) ByID(ctx context.Context, id kernel.ID) (*identity.User, error) {
	row, err := r.q.GetUser(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: user not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

// ByEmail returns a user by email.
func (r *UserRepo) ByEmail(ctx context.Context, email identity.Email) (*identity.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: user not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomainUser(row), nil
}

// Update persists changes to an existing user.
func (r *UserRepo) Update(ctx context.Context, u *identity.User) error {
	n, err := r.q.UpdateUser(ctx, sqlcdb.UpdateUserParams{
		ID:           u.ID.String(),
		Email:        u.Email.String(),
		Role:         userRoleToDB(u.Role),
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Status:       userStatusToDB(u.Status),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return kernel.NotFound("postgres: user not found")
	}
	return nil
}

func toDomainUser(u sqlcdb.User) *identity.User {
	return &identity.User{
		ID:           kernel.ID(u.ID),
		Email:        identity.Email(u.Email),
		Role:         userRoleFromDB(u.Role),
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Status:       userStatusFromDB(u.Status),
		CreatedAt:    u.CreatedAt.Time,
	}
}

func userRoleToDB(r identity.Role) string {
	switch r {
	case identity.RoleEmployer:
		return "employer"
	case identity.RoleRecruiter:
		return "recruiter"
	case identity.RoleCandidate:
		return "candidate"
	default:
		return "unspecified"
	}
}

func userRoleFromDB(s string) identity.Role {
	switch s {
	case "employer":
		return identity.RoleEmployer
	case "recruiter":
		return identity.RoleRecruiter
	case "candidate":
		return identity.RoleCandidate
	default:
		return identity.RoleUnspecified
	}
}

func userStatusToDB(s identity.AccountStatus) string {
	if s == identity.StatusLocked {
		return "locked"
	}
	return "active"
}

func userStatusFromDB(s string) identity.AccountStatus {
	if s == "locked" {
		return identity.StatusLocked
	}
	return identity.StatusActive
}

var _ identity.UserRepository = (*UserRepo)(nil)
