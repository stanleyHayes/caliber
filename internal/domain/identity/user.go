package identity

import (
	"strings"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// AccountStatus is the lifecycle state of a user account.
type AccountStatus int

// Account statuses.
const (
	StatusUnspecified AccountStatus = iota
	StatusActive
	StatusLocked
)

// Valid reports whether the status is known and non-zero.
func (s AccountStatus) Valid() bool { return s == StatusActive || s == StatusLocked }

// String renders the status.
func (s AccountStatus) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusLocked:
		return "locked"
	default:
		return "unspecified"
	}
}

// DefaultPasswordMinLength is the default minimum plaintext password length.
const DefaultPasswordMinLength = 12

// DefaultPasswordMaxLength bounds the plaintext password (CAL-120): Argon2id is
// deliberately CPU/memory-hard, so an unbounded plaintext on the unauthenticated
// Register/Login surface is a cheap amplification DoS. 128 is well above any real
// passphrase and matches common OWASP guidance.
const DefaultPasswordMaxLength = 128

// MaxNameLen bounds a user's display name (CAL-111): untrusted registration input
// that is stored and rendered, so it is length-capped to prevent storage/UI abuse.
const MaxNameLen = 200

// PasswordPolicy validates plaintext passwords before they are hashed.
type PasswordPolicy struct {
	MinLength int
	MaxLength int
}

// DefaultPasswordPolicy returns the standard password policy.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinLength: DefaultPasswordMinLength, MaxLength: DefaultPasswordMaxLength}
}

// MaxAllowedLength returns the effective maximum plaintext password length,
// applying the default when the policy leaves MaxLength unset.
func (p PasswordPolicy) MaxAllowedLength() int {
	if p.MaxLength <= 0 {
		return DefaultPasswordMaxLength
	}
	return p.MaxLength
}

// Validate checks a plaintext password against the policy.
func (p PasswordPolicy) Validate(plain string) error {
	minLen := p.MinLength
	if minLen <= 0 {
		minLen = DefaultPasswordMinLength
	}
	maxLen := p.MaxLength
	if maxLen <= 0 {
		maxLen = DefaultPasswordMaxLength
	}
	if strings.TrimSpace(plain) == "" {
		return kernel.Invalid("identity: password must not be blank")
	}
	if len(plain) < minLen {
		return kernel.Invalidf("identity: password must be at least %d characters", minLen)
	}
	if len(plain) > maxLen {
		return kernel.Invalidf("identity: password must be at most %d characters", maxLen)
	}
	return nil
}

// User is an authenticated principal owning an employer or candidate context.
// PasswordHash holds an already-hashed credential (hashing is an adapter concern).
type User struct {
	ID           kernel.ID
	Email        Email
	Role         Role
	Name         string
	PasswordHash string
	Status       AccountStatus
	CreatedAt    time.Time
}

// NewUser builds a validated, active user.
func NewUser(email Email, role Role, name, passwordHash string, createdAt time.Time) (*User, error) {
	if email == "" {
		return nil, kernel.Invalid("identity: email is required")
	}
	if !role.Valid() {
		return nil, kernel.Invalid("identity: a valid role is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, kernel.Invalid("identity: name is required")
	}
	if len([]rune(name)) > MaxNameLen {
		return nil, kernel.Invalidf("identity: name exceeds %d characters", MaxNameLen)
	}
	if strings.TrimSpace(passwordHash) == "" {
		return nil, kernel.Invalid("identity: password hash is required")
	}
	return &User{
		ID:           kernel.NewID(),
		Email:        email,
		Role:         role,
		Name:         name,
		PasswordHash: passwordHash,
		Status:       StatusActive,
		CreatedAt:    createdAt,
	}, nil
}

// Lock disables the account.
func (u *User) Lock() { u.Status = StatusLocked }

// IsActive reports whether the account is active.
func (u *User) IsActive() bool { return u.Status == StatusActive }
