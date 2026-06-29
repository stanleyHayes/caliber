package memory

import (
	"context"
	"sync"

	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// UserRepo is an in-memory identity.UserRepository for local/dev runs.
type UserRepo struct {
	mu      sync.RWMutex
	byID    map[kernel.ID]identity.User
	byEmail map[identity.Email]kernel.ID
}

// NewUserRepo builds an empty in-memory user repository.
func NewUserRepo() *UserRepo {
	return &UserRepo{byID: map[kernel.ID]identity.User{}, byEmail: map[identity.Email]kernel.ID{}}
}

// Create inserts a new user, rejecting a duplicate email.
func (r *UserRepo) Create(_ context.Context, u *identity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, taken := r.byEmail[u.Email]; taken {
		return kernel.Conflict("memory: user email already exists")
	}
	r.byID[u.ID] = *u
	r.byEmail[u.Email] = u.ID
	return nil
}

// ByID returns a copy of the user with the given id.
func (r *UserRepo) ByID(_ context.Context, id kernel.ID) (*identity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, kernel.NotFound("memory: user not found")
	}
	return &u, nil
}

// ByEmail returns a copy of the user with the given email.
func (r *UserRepo) ByEmail(_ context.Context, email identity.Email) (*identity.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[email]
	if !ok {
		return nil, kernel.NotFound("memory: user not found")
	}
	u := r.byID[id]
	return &u, nil
}

// Update replaces an existing user, keeping the email index consistent.
func (r *UserRepo) Update(_ context.Context, u *identity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.byID[u.ID]
	if !ok {
		return kernel.NotFound("memory: user not found")
	}
	if existing.Email != u.Email {
		delete(r.byEmail, existing.Email)
		r.byEmail[u.Email] = u.ID
	}
	r.byID[u.ID] = *u
	return nil
}

// Anonymize de-identifies a user account in place (right-to-erasure cascade,
// CAL-118): the PII (name, email) and the credential are scrubbed while the row
// is kept so audit/foreign references do not dangle. The email is replaced with a
// unique tombstone so the email index stays valid. Idempotent for an absent id.
func (r *UserRepo) Anonymize(_ context.Context, id kernel.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok {
		return nil
	}
	delete(r.byEmail, u.Email)
	u.Name = ""
	u.PasswordHash = ""
	u.Email = identity.Email("erased+" + id.String() + "@erased.invalid")
	r.byID[id] = u
	r.byEmail[u.Email] = id
	return nil
}

var _ identity.UserRepository = (*UserRepo)(nil)
