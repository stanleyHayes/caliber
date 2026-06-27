// Package salary provides a small, deterministic Ghana-market salary lookup
// (CAL-039). It exists for realism: when a generated Role Spec omits compensation,
// the platform fills a plausible monthly-GHS band from local market data rather
// than leaving it blank. The table is intentionally coarse — a few role families
// scaled across seniority — not a compensation benchmark.
package salary

import (
	"math"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// currency is the denomination of every band returned by Lookup. The platform
// targets Ghana, so market bands are monthly gross GHS (matching the seed data).
const currency = "GHS"

// roundTo is the granularity bands are rounded to, for tidy, credible figures.
const roundTo = 500

// Lookup returns a plausible Ghana-market monthly-GHS salary band for a role,
// classified from its title and scaled by seniority. It always returns a usable
// band (an unrecognised seniority is treated as mid-level), so callers can use it
// as an unconditional fallback when no band is otherwise known.
func Lookup(title string, seniority role.Seniority) kernel.SalaryBand {
	low, high := baseBand(seniority)
	factor := familyFactor(title)
	return kernel.SalaryBand{
		Currency: currency,
		Low:      round(low * factor),
		High:     round(high * factor),
	}
}

// baseBand is the general-engineering monthly-GHS band (low, high) for a
// seniority level. These bracket the seeded demo roles (senior backend ~12k–22k).
func baseBand(seniority role.Seniority) (float64, float64) {
	switch seniority {
	case role.SeniorityJunior:
		return 4000, 8000
	case role.SeniorityMid:
		return 8000, 15000
	case role.SenioritySenior:
		return 13000, 22000
	case role.SeniorityLead:
		return 20000, 34000
	case role.SeniorityUnspecified:
		return 8000, 15000 // unspecified -> mid baseline
	default:
		return 8000, 15000
	}
}

// familyFactor scales the base (general-engineering) band for the role family
// implied by the title, relative to engineering (1.0). Data/ML and platform/SRE
// command a premium in the Ghana tech market; design and QA sit slightly below.
func familyFactor(title string) float64 {
	t := strings.ToLower(title)
	switch {
	case containsAny(t, "data", "machine learning", "ml engineer", "scientist", "analytics"):
		return 1.15
	case containsAny(t, "devops", "sre", "platform", "infrastructure", "reliability", "cloud"):
		return 1.15
	case containsAny(t, "design", "ux", "ui/ux"):
		return 0.85
	case containsAny(t, "qa", "quality assurance", "sdet", "tester"):
		return 0.85
	default:
		return 1.0
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

func round(x float64) float64 { return math.Round(x/roundTo) * roundTo }
