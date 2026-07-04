package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// txBeginner is the subset of *pgxpool.Pool used to open a transaction. The
// UserRepo needs it so that a user and its employer row are written atomically;
// a plain DBTX (e.g. an already-open tx) cannot begin a nested one.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// UserRepo is a Postgres-backed identity.UserRepository.
type UserRepo struct {
	q  *sqlcdb.Queries
	tx txBeginner // non-nil when the underlying handle is a pool (can open a tx)
}

// NewUserRepo builds the repository from a sqlc DBTX. When db is a connection
// pool it is also used to run the two-statement employer creation atomically.
func NewUserRepo(db sqlcdb.DBTX) *UserRepo {
	repo := &UserRepo{q: sqlcdb.New(db)}
	if b, ok := db.(txBeginner); ok {
		repo.tx = b
	}
	return repo
}

// Create inserts a new user. Employer users also get a corresponding employers row
// so that downstream role/match repositories can enforce their foreign-key
// constraints without the caller needing to manage two aggregates. The two writes
// run in a single transaction so a failure of the second cannot leave an orphaned
// user with no employers row — which, because the email is unique, would
// permanently wedge that account (re-registration would return Conflict and never
// retry the employer insert).
func (r *UserRepo) Create(ctx context.Context, u *identity.User) error {
	// A non-employer user is a single insert; no transaction needed.
	if u.Role != identity.RoleEmployer {
		return insertUser(ctx, r.q, u)
	}
	// Without a pool we cannot open a transaction (e.g. the handle is already a
	// tx); fall back to sequential inserts rather than failing outright.
	if r.tx == nil {
		if err := insertUser(ctx, r.q, u); err != nil {
			return err
		}
		return createEmployer(ctx, r.q, u)
	}
	tx, err := r.tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit
	q := r.q.WithTx(tx)
	if err := insertUser(ctx, q, u); err != nil {
		return err
	}
	if err := createEmployer(ctx, q, u); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// insertUser writes the users row, mapping a unique-constraint violation to a
// domain Conflict. It runs against whichever Queries handle (pool or tx) is given.
func insertUser(ctx context.Context, q *sqlcdb.Queries, u *identity.User) error {
	err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
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

// createEmployer writes the employers row mirroring an employer user.
func createEmployer(ctx context.Context, q *sqlcdb.Queries, u *identity.User) error {
	return q.CreateEmployer(ctx, sqlcdb.CreateEmployerParams{
		ID:            u.ID.String(),
		CompanyName:   u.Name,
		ContactUserID: pgtype.Text{String: u.ID.String(), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: u.CreatedAt, Valid: true},
	})
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
	case identity.RoleAdmin:
		return "admin"
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
	case "admin":
		return identity.RoleAdmin
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

// Anonymize scrubs a user's identifying fields in place (CAL-118 right-to-
// erasure). The row is retained — it may still be referenced — but the name and
// password are blanked and the email is replaced with a unique, non-routable
// tombstone so the unique constraint is preserved and no PII remains.
func (r *UserRepo) Anonymize(ctx context.Context, id kernel.ID) error {
	return r.q.AnonymizeUser(ctx, sqlcdb.AnonymizeUserParams{
		ID:    id.String(),
		Email: "erased+" + id.String() + "@erased.invalid",
	})
}

// ListEligibleForRetention returns the ids of candidate-role users whose account
// was created on or before the cutoff — the subjects whose data has aged past the
// retention window (CAL-158). A candidate's id equals their user id.
func (r *UserRepo) ListEligibleForRetention(ctx context.Context, createdOnOrBefore time.Time) ([]kernel.ID, error) {
	rows, err := r.q.ListRetentionEligibleCandidates(ctx, pgtype.Timestamptz{Time: createdOnOrBefore, Valid: true})
	if err != nil {
		return nil, err
	}
	ids := make([]kernel.ID, len(rows))
	for i, id := range rows {
		ids[i] = kernel.ID(id)
	}
	return ids, nil
}

var _ identity.UserRepository = (*UserRepo)(nil)
