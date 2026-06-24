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
)

// Valid reports whether the role is a known, non-zero role.
func (r Role) Valid() bool { return r >= RoleEmployer && r <= RoleCandidate }

// String renders the role.
func (r Role) String() string {
	switch r {
	case RoleEmployer:
		return "employer"
	case RoleRecruiter:
		return "recruiter"
	case RoleCandidate:
		return "candidate"
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
	default:
		return RoleUnspecified, kernel.Invalidf("identity: unknown role %q", s)
	}
}
