// Package identity holds the authentication use-cases: register, login, token
// refresh (with rotation), and logout. It orchestrates the domain user
// repository and the password/token/refresh-store ports.
package identity

import (
	"context"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	identitydom "github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Session is the result of an authentication: the user and a fresh token pair.
type Session struct {
	User    *identitydom.User
	Access  app.AccessToken
	Refresh app.RefreshToken
}

// RegisterInput is the input to Register.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Role     identitydom.Role
}

// Provisioner bootstraps the bounded-context aggregate a newly registered user
// owns (e.g. a candidate's Talent Passport). It is invoked during Register after
// the user is persisted. A nil-role-match implementation should be a no-op.
type Provisioner interface {
	Provision(ctx context.Context, user *identitydom.User) error
}

// Service implements the identity use-cases over the domain and security ports.
type Service struct {
	users       identitydom.UserRepository
	hasher      app.PasswordHasher
	tokens      app.TokenService
	refresh     app.RefreshTokenStore
	now         app.Clock
	policy      identitydom.PasswordPolicy
	provisioner Provisioner
	throttle    app.LoginThrottle
}

// Option customizes a Service.
type Option func(*Service)

// WithProvisioner installs a context-bootstrap provisioner invoked on Register.
func WithProvisioner(p Provisioner) Option {
	return func(s *Service) { s.provisioner = p }
}

// WithThrottle installs a brute-force login throttle.
func WithThrottle(t app.LoginThrottle) Option {
	return func(s *Service) { s.throttle = t }
}

// NewService wires the use-case. A nil clock defaults to time.Now.
func NewService(
	users identitydom.UserRepository,
	hasher app.PasswordHasher,
	tokens app.TokenService,
	refresh app.RefreshTokenStore,
	now app.Clock,
	opts ...Option,
) *Service {
	if now == nil {
		now = time.Now
	}
	s := &Service{
		users: users, hasher: hasher, tokens: tokens, refresh: refresh,
		now: now, policy: identitydom.DefaultPasswordPolicy(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Register validates the input, creates the user with a hashed password, and
// issues a session. A duplicate email surfaces as a kernel.Conflict.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*Session, error) {
	email, err := identitydom.NewEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if err := s.policy.Validate(in.Password); err != nil {
		return nil, err
	}
	if !in.Role.Valid() {
		return nil, kernel.Invalid("identity: a valid role is required")
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	user, err := identitydom.NewUser(email, in.Role, in.Name, hash, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	if s.provisioner != nil {
		// Best-effort atomicity is not available across two repositories without
		// a shared transaction; a provisioning failure fails the registration so
		// the partial state surfaces rather than silently leaving a context-less
		// account.
		if err := s.provisioner.Provision(ctx, user); err != nil {
			return nil, err
		}
	}
	return s.issue(ctx, user)
}

// Login verifies credentials and issues a session. To avoid account enumeration,
// an unknown email, a wrong password, and an inactive account all return the
// same generic unauthorized error.
func (s *Service) Login(ctx context.Context, rawEmail, password string) (*Session, error) {
	email, err := identitydom.NewEmail(rawEmail)
	if err != nil {
		return nil, invalidCredentials()
	}
	key := email.String()
	if err := s.throttleCheck(ctx, key); err != nil {
		return nil, err
	}
	user, err := s.verifyCredentials(ctx, email, password)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindUnauthorized {
			s.throttleFail(ctx, key)
		}
		return nil, err
	}
	s.throttleReset(ctx, key)
	return s.issue(ctx, user)
}

// Refresh rotates a refresh token: it verifies the token, single-use-consumes
// its grant (rejecting replays), and issues a fresh session.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	claims, err := s.tokens.VerifyRefresh(refreshToken)
	if err != nil {
		return nil, err
	}
	rec, err := s.refresh.Consume(ctx, claims.ID, s.now())
	if err != nil {
		return nil, err
	}
	user, err := s.users.ByID(ctx, rec.UserID)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindNotFound {
			return nil, kernel.Unauthorized("identity: account no longer exists")
		}
		return nil, err
	}
	if !user.IsActive() {
		return nil, kernel.Unauthorized("identity: account is not active")
	}
	return s.issue(ctx, user)
}

// Logout revokes the refresh grant. It is idempotent: an invalid or unknown
// token is treated as already logged out.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokens.VerifyRefresh(refreshToken)
	if err != nil {
		return nil //nolint:nilerr // idempotent logout: an invalid token has nothing to revoke
	}
	return s.refresh.Revoke(ctx, claims.ID)
}

// Me returns the user behind an authenticated principal.
func (s *Service) Me(ctx context.Context, userID kernel.ID) (*identitydom.User, error) {
	return s.users.ByID(ctx, userID)
}

// verifyCredentials loads the user and checks the password, returning a generic
// unauthorized error for an unknown email, a wrong password, or an inactive
// account. An unknown email still performs equivalent hashing work so response
// time does not reveal whether the account exists (enumeration defense).
func (s *Service) verifyCredentials(ctx context.Context, email identitydom.Email, password string) (*identitydom.User, error) {
	user, err := s.users.ByEmail(ctx, email)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindNotFound {
			_, _ = s.hasher.Hash(password) // equalize timing against the verify path
			return nil, invalidCredentials()
		}
		return nil, err
	}
	ok, err := s.hasher.Verify(user.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	if !ok || !user.IsActive() {
		return nil, invalidCredentials()
	}
	return user, nil
}

func (s *Service) throttleCheck(ctx context.Context, key string) error {
	if s.throttle == nil {
		return nil
	}
	return s.throttle.Check(ctx, key)
}

func (s *Service) throttleFail(ctx context.Context, key string) {
	if s.throttle != nil {
		s.throttle.Fail(ctx, key)
	}
}

func (s *Service) throttleReset(ctx context.Context, key string) {
	if s.throttle != nil {
		s.throttle.Reset(ctx, key)
	}
}

// issue mints an access + refresh pair and records the refresh grant.
func (s *Service) issue(ctx context.Context, user *identitydom.User) (*Session, error) {
	p := app.Principal{UserID: user.ID, Role: user.Role.String()}
	access, err := s.tokens.IssueAccess(p)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.IssueRefresh(p)
	if err != nil {
		return nil, err
	}
	rec := app.RefreshRecord{ID: refresh.ID, UserID: user.ID, ExpiresAt: s.now().Add(refresh.ExpiresIn)}
	if err := s.refresh.Save(ctx, rec); err != nil {
		return nil, err
	}
	return &Session{User: user, Access: access, Refresh: refresh}, nil
}

func invalidCredentials() error { return kernel.Unauthorized("identity: invalid email or password") }
