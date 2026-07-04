// Package identity holds authentication rules and the two application roles
// (hiring side: employer/recruiter, and candidate). Pure domain on the kernel.
package identity

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Role is the kind of account a user holds.
type Role int

// Account roles.
const (
	RoleUnspecified Role = iota
	RoleEmployer
	RoleRecruiter
	RoleCandidate
	// RoleAdmin is a platform operator (CAL-154). It is a valid role for
	// authentication/RBAC but is NOT self-registerable — admins are provisioned
	// out-of-band, never through public registration (see Registerable).
	RoleAdmin
)

// Valid reports whether the role is a known, non-zero role.
func (r Role) Valid() bool { return r >= RoleEmployer && r <= RoleAdmin }

// Registerable reports whether a user may self-register with this role via the
// public Register flow. Admin is excluded (provisioned out-of-band) so a caller
// cannot create a platform-operator account for themselves.
func (r Role) Registerable() bool {
	return r == RoleEmployer || r == RoleRecruiter || r == RoleCandidate
}

// String renders the role.
func (r Role) String() string {
	switch r {
	case RoleEmployer:
		return "employer"
	case RoleRecruiter:
		return "recruiter"
	case RoleCandidate:
		return "candidate"
	case RoleAdmin:
		return "admin"
	default:
		return "unspecified"
	}
}

// ParseRole converts a string (case-insensitive) into a Role.
func ParseRole(s string) (Role, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "employer":
		return RoleEmployer, nil
	case "recruiter":
		return RoleRecruiter, nil
	case "candidate":
		return RoleCandidate, nil
	case "admin":
		return RoleAdmin, nil
	default:
		return RoleUnspecified, kernel.Invalidf("identity: unknown role %q", s)
	}
}
