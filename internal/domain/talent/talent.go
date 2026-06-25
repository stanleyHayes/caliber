// Package talent is the Talent Passport bounded context: it models candidates,
// their intake preferences, and the talent profile (competencies and passport
// status) derived from CV ingestion, screening, and verification.
//
// It is a pure domain package: it depends only on the shared kernel and the Go
// standard library, and never on sibling domain packages, application, or
// adapter code.
package talent

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// PassportStatus is the lifecycle stage of a talent passport, from a raw CV
// upload through screening to full verification.
type PassportStatus int

// Talent passport lifecycle stages. The zero value is an invalid/unset status.
const (
	PassportUnset PassportStatus = iota
	PassportCVOnly
	PassportScreened
	PassportVerified
)

// Valid reports whether the status is a known, non-zero stage.
func (s PassportStatus) Valid() bool {
	return s >= PassportCVOnly && s <= PassportVerified
}

// String renders the passport status.
func (s PassportStatus) String() string {
	switch s {
	case PassportCVOnly:
		return "cv_only"
	case PassportScreened:
		return "screened"
	case PassportVerified:
		return "verified"
	default:
		return "unset"
	}
}

// ProfileCompetency is a single evidenced competency on a talent profile: a
// named skill with a level in [0,5] backed by an evidence quote and the source
// span it was extracted from.
type ProfileCompetency struct {
	Name          string
	Level         float64
	EvidenceQuote string
	SourceSpan    string
}

// Validate checks the competency has a non-empty name and a level in [0,5].
func (c ProfileCompetency) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return kernel.Invalid("competency name is required")
	}
	if c.Level < 0 || c.Level > 5 {
		return kernel.Invalidf("competency level must be in [0,5], got %v", c.Level)
	}
	return nil
}

// CandidateIntake captures a candidate's job-search preferences gathered at
// intake: desired titles, location, salary floor, and deal-breakers.
type CandidateIntake struct {
	TargetTitles   []string
	Location       string
	SalaryFloor    float64
	SalaryCurrency string
	DealBreakers   []string
}

// Validate checks the intake is internally consistent (non-negative salary floor).
func (i CandidateIntake) Validate() error {
	if i.SalaryFloor < 0 {
		return kernel.Invalid("salary floor must be non-negative")
	}
	return nil
}

// Candidate is a person seeking roles, linked to a platform user and carrying
// their intake preferences.
type Candidate struct {
	ID       kernel.ID
	UserID   kernel.ID
	Location string
	Intake   CandidateIntake
}

// NewCandidate validates inputs and constructs a Candidate with a fresh ID.
// userID must be non-zero and the intake must be valid.
func NewCandidate(userID kernel.ID, location string, intake CandidateIntake) (*Candidate, error) {
	if userID.IsZero() {
		return nil, kernel.Invalid("candidate user id is required")
	}
	if err := intake.Validate(); err != nil {
		return nil, err
	}
	return &Candidate{
		ID:       kernel.NewID(),
		UserID:   userID,
		Location: location,
		Intake:   intake,
	}, nil
}

// TalentProfile is the passport for a candidate: a summary, a set of evidenced
// competencies, and a verification status.
type TalentProfile struct { //nolint:revive // name fixed by domain spec (Talent Passport context)
	ID             kernel.ID
	CandidateID    kernel.ID
	Summary        string
	Competencies   []ProfileCompetency
	PassportStatus PassportStatus
}

// NewTalentProfile validates inputs and constructs a TalentProfile with a fresh
// ID and PassportCVOnly status. candidateID must be non-zero and every supplied
// competency must be valid.
func NewTalentProfile(candidateID kernel.ID, summary string, comps []ProfileCompetency) (*TalentProfile, error) {
	if candidateID.IsZero() {
		return nil, kernel.Invalid("talent profile candidate id is required")
	}
	for i := range comps {
		if err := comps[i].Validate(); err != nil {
			return nil, err
		}
	}
	cloned := make([]ProfileCompetency, len(comps))
	copy(cloned, comps)
	return &TalentProfile{
		ID:             kernel.NewID(),
		CandidateID:    candidateID,
		Summary:        summary,
		Competencies:   cloned,
		PassportStatus: PassportCVOnly,
	}, nil
}

// MarkScreened advances the passport to the screened status.
func (p *TalentProfile) MarkScreened() { p.PassportStatus = PassportScreened }

// MarkVerified advances the passport to the verified status.
func (p *TalentProfile) MarkVerified() { p.PassportStatus = PassportVerified }

// AddCompetency validates and appends a competency to the profile.
func (p *TalentProfile) AddCompetency(c ProfileCompetency) error {
	if err := c.Validate(); err != nil {
		return err
	}
	p.Competencies = append(p.Competencies, c)
	return nil
}
