// Package role holds the Role Spec and weighted Rubric domain (Flow A.1).
package role

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Seniority is the seniority level of a role.
type Seniority int

// Seniority levels.
const (
	SeniorityUnspecified Seniority = iota
	SeniorityJunior
	SeniorityMid
	SenioritySenior
	SeniorityLead
)

// Valid reports whether the seniority is known and non-zero.
func (s Seniority) Valid() bool { return s >= SeniorityJunior && s <= SeniorityLead }

// String renders the seniority.
func (s Seniority) String() string {
	switch s {
	case SeniorityJunior:
		return "junior"
	case SeniorityMid:
		return "mid"
	case SenioritySenior:
		return "senior"
	case SeniorityLead:
		return "lead"
	default:
		return "unspecified"
	}
}

// ParseSeniority converts a string (case-insensitive) into a Seniority.
func ParseSeniority(s string) (Seniority, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "junior":
		return SeniorityJunior, nil
	case "mid":
		return SeniorityMid, nil
	case "senior":
		return SenioritySenior, nil
	case "lead":
		return SeniorityLead, nil
	default:
		return SeniorityUnspecified, kernel.Invalidf("role: unknown seniority %q", s)
	}
}
