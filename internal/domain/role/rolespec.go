package role

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RoleSpec is a structured representation of an open role (Appendix A.1).
type RoleSpec struct { //nolint:revive // domain name fixed by the role context spec
	Title            string
	Location         string
	Seniority        Seniority
	Availability     string
	Responsibilities []string
	MustHaves        []string
	NiceToHaves      []string
	SalaryBand       kernel.SalaryBand
}

// Validate checks the spec is well-formed.
func (s RoleSpec) Validate() error {
	if strings.TrimSpace(s.Title) == "" {
		return kernel.Invalid("role: title is required")
	}
	if !s.Seniority.Valid() {
		return kernel.Invalid("role: a valid seniority is required")
	}
	return s.SalaryBand.Validate()
}
