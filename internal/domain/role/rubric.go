package role

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// weightEpsilon is the tolerance for rubric weights summing to 1.0.
const weightEpsilon = 0.01

// Competency is a single weighted, scoreable competency in a rubric.
type Competency struct {
	Name     string
	Weight   float64
	MustHave bool
}

// Validate checks the competency is well-formed.
func (c Competency) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return kernel.Invalid("role: competency name is required")
	}
	if c.Weight < 0 || c.Weight > 1 {
		return kernel.Invalidf("role: competency %q weight must be in [0,1]", c.Name)
	}
	return nil
}

// Rubric is the weighted set of competencies a role is scored against.
type Rubric struct {
	Competencies []Competency
}

// TotalWeight returns the sum of competency weights.
func (r Rubric) TotalWeight() float64 {
	var sum float64
	for _, c := range r.Competencies {
		sum += c.Weight
	}
	return sum
}

// Validate checks the rubric has at least one valid competency whose weights
// sum to within weightEpsilon of 1.0.
func (r Rubric) Validate() error {
	if len(r.Competencies) == 0 {
		return kernel.Invalid("role: rubric must have at least one competency")
	}
	for _, c := range r.Competencies {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	sum := r.TotalWeight()
	if sum < 1-weightEpsilon || sum > 1+weightEpsilon {
		return kernel.Invalidf("role: rubric weights must sum to 1.0 (got %.4f)", sum)
	}
	return nil
}

// Normalize returns a copy of the rubric with weights scaled to sum to 1.0.
// If the total weight is zero the rubric is returned unchanged.
func (r Rubric) Normalize() Rubric {
	sum := r.TotalWeight()
	if sum == 0 {
		return r
	}
	out := make([]Competency, len(r.Competencies))
	for i, c := range r.Competencies {
		c.Weight /= sum
		out[i] = c
	}
	return Rubric{Competencies: out}
}
