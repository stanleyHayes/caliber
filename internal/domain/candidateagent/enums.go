package candidateagent

// ApplicationSource identifies who authored an application: a human candidate
// acting manually, or the autonomous candidate agent.
type ApplicationSource int

// Application source values. The zero value is intentionally invalid so an
// unset source is detectable and rejected by validation.
const (
	// SourceUnknown is the zero value and is never a valid source.
	SourceUnknown ApplicationSource = iota
	// SourceManual marks an application authored manually by the candidate.
	SourceManual
	// SourceAgent marks an application authored by the autonomous candidate agent.
	SourceAgent
)

// Valid reports whether the source is a known, non-zero value.
func (s ApplicationSource) Valid() bool {
	switch s {
	case SourceManual, SourceAgent:
		return true
	default:
		return false
	}
}

// String returns a stable, human-readable label for the source.
func (s ApplicationSource) String() string {
	switch s {
	case SourceManual:
		return "manual"
	case SourceAgent:
		return "agent"
	default:
		return "unknown"
	}
}

// ApplicationStatus is the lifecycle state of an application as it moves from a
// draft through submission and screening.
type ApplicationStatus int

// Application status values. The zero value is intentionally invalid so an
// unset status is detectable and rejected by validation.
const (
	// StatusUnknown is the zero value and is never a valid status.
	StatusUnknown ApplicationStatus = iota
	// StatusDrafted is the initial state of a freshly created application.
	StatusDrafted
	// StatusSubmitted means the application has been submitted to the employer.
	StatusSubmitted
	// StatusScreening means the application is actively being screened.
	StatusScreening
	// StatusScreened means screening of the application has completed.
	StatusScreened
)

// Valid reports whether the status is a known, non-zero value.
func (s ApplicationStatus) Valid() bool {
	switch s {
	case StatusDrafted, StatusSubmitted, StatusScreening, StatusScreened:
		return true
	default:
		return false
	}
}

// String returns a stable, human-readable label for the status.
func (s ApplicationStatus) String() string {
	switch s {
	case StatusDrafted:
		return "drafted"
	case StatusSubmitted:
		return "submitted"
	case StatusScreening:
		return "screening"
	case StatusScreened:
		return "screened"
	default:
		return "unknown"
	}
}
