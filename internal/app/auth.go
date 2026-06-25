package app

import (
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
