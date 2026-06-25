package app

import (
	"context"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

//go:generate mockgen -source=auth.go -destination=../mocks/auth.go -package=mocks

// PasswordHasher hashes plaintext passwords and verifies candidates against an
// encoded hash (CAL-018). Implementations must be timing-safe on verification.
type PasswordHasher interface {
	// Hash returns an encoded hash (salt + parameters embedded) for a plaintext.
	Hash(plain string) (string, error)
	// Verify reports whether plain matches the encoded hash. A malformed hash
	// returns an error; a non-match returns (false, nil).
	Verify(encodedHash, plain string) (bool, error)
}

// Principal is the authenticated subject carried inside a verified token.
type Principal struct {
	UserID kernel.ID
	Role   string // identity.Role rendered as a string (e.g. "employer")
}

// AccessToken is a signed short-lived access token and its lifetime.
type AccessToken struct {
	Token     string
	ExpiresIn time.Duration
}

// RefreshToken is a signed rotating refresh token, its server-side identifier
// (jti, for revocation/rotation bookkeeping), and its lifetime.
type RefreshToken struct {
	Token     string
	ID        string
	ExpiresIn time.Duration
}

// RefreshClaims is the verified content of a refresh token.
type RefreshClaims struct {
	Principal Principal
	ID        string // jti
}

// TokenService issues and verifies access and refresh tokens (CAL-019). Access
// tokens are stateless; refresh tokens carry a jti so the caller can rotate and
// revoke them through a server-side store.
type TokenService interface {
	IssueAccess(p Principal) (AccessToken, error)
	IssueRefresh(p Principal) (RefreshToken, error)
	VerifyAccess(token string) (Principal, error)
	VerifyRefresh(token string) (RefreshClaims, error)
}

// RefreshRecord is a persisted refresh-token grant, keyed by its jti.
type RefreshRecord struct {
	ID        string // jti
	UserID    kernel.ID
	ExpiresAt time.Time
	Revoked   bool
}

// RefreshTokenStore tracks issued refresh tokens so they can be rotated and
// revoked (CAL-019/020). Refresh tokens are single-use: a successful Consume
// revokes the record, so a replayed token is rejected (replay detection).
type RefreshTokenStore interface {
	// Save records a freshly issued refresh grant.
	Save(ctx context.Context, rec RefreshRecord) error
	// Consume atomically validates and revokes a grant by jti. It returns the
	// record on success, or a kernel.KindUnauthorized error when the jti is
	// unknown, already consumed/revoked, or expired.
	Consume(ctx context.Context, jti string, now time.Time) (RefreshRecord, error)
	// Revoke marks a grant revoked (logout). Revoking an unknown jti is a no-op.
	Revoke(ctx context.Context, jti string) error
}

// LoginThrottle protects the login path from brute-force / credential-stuffing
// by limiting failed attempts per key (e.g. normalized email). It is a
// best-effort, in-process control for the POC.
type LoginThrottle interface {
	// Check reports a kernel.KindTooManyRequests error when the key is currently
	// locked out, or nil when a login attempt may proceed.
	Check(ctx context.Context, key string) error
	// Fail records a failed attempt for the key.
	Fail(ctx context.Context, key string)
	// Reset clears the attempt counter for the key after a successful login.
	Reset(ctx context.Context, key string)
}
