package auth

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
	signingMethod    = "HS256"

	// minSecretBytes is the floor for an HS256 signing key (256-bit). HS256
	// security reduces entirely to secret entropy: a captured token is an
	// offline brute-force oracle, so a weak secret means forgeable tokens.
	minSecretBytes = 32
)

// Default token lifetimes.
const (
	DefaultAccessTTL  = 15 * time.Minute
	DefaultRefreshTTL = 7 * 24 * time.Hour
)

// JWTConfig configures the HS256 token service.
type JWTConfig struct {
	Secret     string
	Issuer     string
	Audience   string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// JWTService implements app.TokenService with HS256-signed JWTs. Access tokens
// are stateless; refresh tokens carry a jti for server-side rotation.
type JWTService struct {
	cfg JWTConfig
	now func() time.Time
}

// JWTOption customizes the service.
type JWTOption func(*JWTService)

// WithClock injects a clock (deterministic tests).
func WithClock(now func() time.Time) JWTOption {
	return func(s *JWTService) { s.now = now }
}

// NewJWTService validates config, applies defaults, and builds the service.
func NewJWTService(cfg JWTConfig, opts ...JWTOption) (*JWTService, error) {
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, kernel.Invalid("auth: JWT secret is required")
	}
	if len(cfg.Secret) < minSecretBytes {
		return nil, kernel.Invalidf("auth: JWT secret must be at least %d bytes", minSecretBytes)
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "caliber"
	}
	if cfg.Audience == "" {
		cfg.Audience = "caliber-api"
	}
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = DefaultAccessTTL
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = DefaultRefreshTTL
	}
	s := &JWTService{cfg: cfg, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

type claims struct {
	jwt.RegisteredClaims

	Role string `json:"role"`
	Type string `json:"typ"`
}

// IssueAccess signs a short-lived access token for the principal.
func (s *JWTService) IssueAccess(p app.Principal) (app.AccessToken, error) {
	jti, err := newID()
	if err != nil {
		return app.AccessToken{}, err
	}
	token, err := s.sign(p, tokenTypeAccess, jti, s.cfg.AccessTTL)
	if err != nil {
		return app.AccessToken{}, err
	}
	return app.AccessToken{Token: token, ExpiresIn: s.cfg.AccessTTL}, nil
}

// IssueRefresh signs a rotating refresh token, returning its jti.
func (s *JWTService) IssueRefresh(p app.Principal) (app.RefreshToken, error) {
	jti, err := newID()
	if err != nil {
		return app.RefreshToken{}, err
	}
	token, err := s.sign(p, tokenTypeRefresh, jti, s.cfg.RefreshTTL)
	if err != nil {
		return app.RefreshToken{}, err
	}
	return app.RefreshToken{Token: token, ID: jti, ExpiresIn: s.cfg.RefreshTTL}, nil
}

// VerifyAccess parses and validates an access token, returning its principal.
func (s *JWTService) VerifyAccess(token string) (app.Principal, error) {
	c, err := s.parse(token)
	if err != nil {
		return app.Principal{}, err
	}
	if c.Type != tokenTypeAccess {
		return app.Principal{}, kernel.Unauthorized("auth: not an access token")
	}
	return app.Principal{UserID: kernel.ID(c.Subject), Role: c.Role}, nil
}

// VerifyRefresh parses and validates a refresh token, returning its principal
// and jti for rotation/revocation checks.
func (s *JWTService) VerifyRefresh(token string) (app.RefreshClaims, error) {
	c, err := s.parse(token)
	if err != nil {
		return app.RefreshClaims{}, err
	}
	if c.Type != tokenTypeRefresh {
		return app.RefreshClaims{}, kernel.Unauthorized("auth: not a refresh token")
	}
	return app.RefreshClaims{
		Principal: app.Principal{UserID: kernel.ID(c.Subject), Role: c.Role},
		ID:        c.ID,
	}, nil
}

func (s *JWTService) sign(p app.Principal, typ, jti string, ttl time.Duration) (string, error) {
	now := s.now()
	c := claims{
		Role: p.Role,
		Type: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.UserID.String(),
			Issuer:    s.cfg.Issuer,
			Audience:  jwt.ClaimStrings{s.cfg.Audience},
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return "", kernel.Wrap(err, kernel.KindInternal, "auth: token signing failed")
	}
	return signed, nil
}

func (s *JWTService) parse(token string) (*claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{},
		func(*jwt.Token) (any, error) { return []byte(s.cfg.Secret), nil },
		jwt.WithValidMethods([]string{signingMethod}),
		jwt.WithIssuer(s.cfg.Issuer),
		jwt.WithAudience(s.cfg.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return nil, kernel.Wrap(err, kernel.KindUnauthorized, "auth: invalid token")
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid {
		return nil, kernel.Unauthorized("auth: invalid token")
	}
	return c, nil
}

// newID returns a 128-bit random hex identifier (jti).
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", kernel.Wrap(err, kernel.KindInternal, "auth: id generation failed")
	}
	return hex.EncodeToString(b), nil
}

var _ app.TokenService = (*JWTService)(nil)
