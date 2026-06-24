package candidateagent

import (
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// EnsureFromProfile enforces the no-fabrication invariant: an agent-authored
// application must be grounded in a verified profile and may draw only on
// verified content. It returns a kernel.Invalid error when profileID is zero,
// signalling that no verified profile was supplied to ground the application.
func EnsureFromProfile(profileID kernel.ID) error {
	if profileID.IsZero() {
		return kernel.Invalid("agent application must reference a verified profile")
	}
	return nil
}

// Application is a candidate's application to a role. It records the authoring
// source and lifecycle status. Agent-authored applications additionally carry a
// non-zero ProfileID enforcing the no-fabrication invariant.
type Application struct {
	// ID is the unique identifier of the application.
	ID kernel.ID
	// RoleID references the role being applied to.
	RoleID kernel.ID
	// CandidateID references the applying candidate.
	CandidateID kernel.ID
	// ProfileID references the verified profile the application draws on. It is
	// always non-zero for agent-authored applications.
	ProfileID kernel.ID
	// Source records whether the application was authored manually or by the agent.
	Source ApplicationSource
	// TailoredSummary is the application summary tailored to the role.
	TailoredSummary string
	// Status is the current lifecycle state of the application.
	Status ApplicationStatus
}

// NewAgentApplication constructs an agent-authored application in the drafted
// state. It enforces the no-fabrication invariant: roleID, candidateID, and
// profileID must be non-zero (profileID via EnsureFromProfile), and the
// tailored summary must be non-empty. The new application is assigned a fresh
// ID, SourceAgent, and StatusDrafted.
func NewAgentApplication(roleID, candidateID, profileID kernel.ID, tailoredSummary string) (*Application, error) {
	if roleID.IsZero() {
		return nil, kernel.Invalid("role id is required")
	}
	if candidateID.IsZero() {
		return nil, kernel.Invalid("candidate id is required")
	}
	if err := EnsureFromProfile(profileID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tailoredSummary) == "" {
		return nil, kernel.Invalid("tailored summary is required")
	}
	return &Application{
		ID:              kernel.NewID(),
		RoleID:          roleID,
		CandidateID:     candidateID,
		ProfileID:       profileID,
		Source:          SourceAgent,
		TailoredSummary: tailoredSummary,
		Status:          StatusDrafted,
	}, nil
}

// NewManualApplication constructs a manually-authored application in the drafted
// state. roleID and candidateID must be non-zero and the tailored summary must
// be non-empty. The new application is assigned a fresh ID, SourceManual, and
// StatusDrafted. ProfileID is left zero because manual applications are not
// bound to a verified profile.
func NewManualApplication(roleID, candidateID kernel.ID, tailoredSummary string) (*Application, error) {
	if roleID.IsZero() {
		return nil, kernel.Invalid("role id is required")
	}
	if candidateID.IsZero() {
		return nil, kernel.Invalid("candidate id is required")
	}
	if strings.TrimSpace(tailoredSummary) == "" {
		return nil, kernel.Invalid("tailored summary is required")
	}
	return &Application{
		ID:              kernel.NewID(),
		RoleID:          roleID,
		CandidateID:     candidateID,
		Source:          SourceManual,
		TailoredSummary: tailoredSummary,
		Status:          StatusDrafted,
	}, nil
}

// Submit transitions the application from drafted to submitted. It returns a
// kernel.Invalid error if the application is not in the drafted state.
func (a *Application) Submit() error {
	if a.Status != StatusDrafted {
		return kernel.Invalidf("cannot submit application in status %q", a.Status)
	}
	a.Status = StatusSubmitted
	return nil
}

// MarkScreening transitions the application from submitted to screening. It
// returns a kernel.Invalid error if the application is not in the submitted state.
func (a *Application) MarkScreening() error {
	if a.Status != StatusSubmitted {
		return kernel.Invalidf("cannot start screening application in status %q", a.Status)
	}
	a.Status = StatusScreening
	return nil
}

// MarkScreened transitions the application from screening to screened. It
// returns a kernel.Invalid error if the application is not in the screening state.
func (a *Application) MarkScreened() error {
	if a.Status != StatusScreening {
		return kernel.Invalidf("cannot complete screening application in status %q", a.Status)
	}
	a.Status = StatusScreened
	return nil
}
