package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RefreshStore is a Postgres-backed app.RefreshTokenStore. Consume flips the
// revoked flag atomically in a single UPDATE ... RETURNING, so a refresh token
// is genuinely single-use even under concurrent rotation attempts.
type RefreshStore struct {
	q *sqlcdb.Queries
}

// NewRefreshStore builds the store from a sqlc DBTX.
func NewRefreshStore(db sqlcdb.DBTX) *RefreshStore { return &RefreshStore{q: sqlcdb.New(db)} }

// Save records a freshly issued refresh grant.
func (s *RefreshStore) Save(ctx context.Context, rec app.RefreshRecord) error {
	return s.q.SaveRefreshToken(ctx, sqlcdb.SaveRefreshTokenParams{
		ID:        rec.ID,
		UserID:    rec.UserID.String(),
		ExpiresAt: pgtype.Timestamptz{Time: rec.ExpiresAt, Valid: true},
	})
}

// Consume atomically validates and revokes a grant by jti. An unknown, already
// consumed/revoked, or expired token yields a kernel.Unauthorized error.
func (s *RefreshStore) Consume(ctx context.Context, jti string, now time.Time) (app.RefreshRecord, error) {
	row, err := s.q.ConsumeRefreshToken(ctx, sqlcdb.ConsumeRefreshTokenParams{
		ID:  jti,
		Now: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return app.RefreshRecord{}, kernel.Unauthorized("auth: refresh token is not valid")
	}
	if err != nil {
		return app.RefreshRecord{}, err
	}
	return app.RefreshRecord{
		ID:        row.ID,
		UserID:    kernel.ID(row.UserID),
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   true,
	}, nil
}

// Revoke marks a grant revoked (logout); an unknown jti is a no-op.
func (s *RefreshStore) Revoke(ctx context.Context, jti string) error {
	return s.q.RevokeRefreshToken(ctx, jti)
}

var _ app.RefreshTokenStore = (*RefreshStore)(nil)
