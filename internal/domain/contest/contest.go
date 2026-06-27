// Package contest is the assessment-contest bounded context (CAL-083): a
// candidate can view and dispute an assessment (a shortlist match or an
// interview report card). Contests are a fairness control — they are explainable
// and auditable, and a human resolves them (human-in-the-loop). Pure domain on
// the shared kernel.
package contest

import (
	"strings"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// Subject is the kind of assessment a contest disputes.
type Subject int

// Contest subjects.
const (
	SubjectUnspecified Subject = iota
	SubjectMatch               // an explainable shortlist match
	SubjectReportCard          // an interview report card
)

// Valid reports whether the subject is a known, non-zero kind.
func (s Subject) Valid() bool { return s == SubjectMatch || s == SubjectReportCard }

// String renders the subject.
func (s Subject) String() string {
	switch s {
	case SubjectMatch:
		return "match"
	case SubjectReportCard:
		return "report_card"
	default:
		return "unspecified"
	}
}

// ParseSubject converts a string (case-insensitive) into a Subject.
func ParseSubject(raw string) (Subject, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "match":
		return SubjectMatch, nil
	case "report_card":
		return SubjectReportCard, nil
	default:
		return SubjectUnspecified, kernel.Invalidf("contest: unknown subject %q", raw)
	}
}

// Status is the lifecycle state of a contest.
type Status int

// Contest statuses. The zero value is the initial open state.
const (
	StatusOpen      Status = iota
	StatusUpheld           // a reviewer agreed with the candidate
	StatusDismissed        // a reviewer disagreed
)

// String renders the status.
func (s Status) String() string {
	switch s {
	case StatusUpheld:
		return "upheld"
	case StatusDismissed:
		return "dismissed"
	default:
		return "open"
	}
}

// MaxReasonLen bounds the candidate-supplied reason.
const MaxReasonLen = 2000

// Contest is a candidate's dispute of an assessment, opened in the open state
// and resolved by a human reviewer.
type Contest struct {
	ID          kernel.ID
	CandidateID kernel.ID
	Subject     Subject
	SubjectID   kernel.ID // the contested match / report-card id
	Reason      string
	Status      Status
	Resolution  string // reviewer's note when resolved
	CreatedAt   time.Time
	ResolvedAt  time.Time
}

// NewContest validates inputs and constructs an open contest with a fresh ID.
// candidateID and subjectID must be non-zero, the subject must be valid, and the
// reason must be non-blank and within MaxReasonLen.
func NewContest(candidateID, subjectID kernel.ID, subject Subject, reason string, now time.Time) (*Contest, error) {
	if candidateID.IsZero() {
		return nil, kernel.Invalid("contest: candidate id is required")
	}
	if subjectID.IsZero() {
		return nil, kernel.Invalid("contest: subject id is required")
	}
	if !subject.Valid() {
		return nil, kernel.Invalid("contest: a valid subject is required")
	}
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return nil, kernel.Invalid("contest: a reason is required")
	}
	if len([]rune(trimmed)) > MaxReasonLen {
		return nil, kernel.Invalidf("contest: reason exceeds %d characters", MaxReasonLen)
	}
	return &Contest{
		ID:          kernel.NewID(),
		CandidateID: candidateID,
		Subject:     subject,
		SubjectID:   subjectID,
		Reason:      trimmed,
		Status:      StatusOpen,
		CreatedAt:   now,
	}, nil
}

// Resolve transitions an open contest to upheld or dismissed, recording the
// reviewer's note and the resolution time. It returns a kernel.Invalid error if
// the contest is already resolved.
func (c *Contest) Resolve(upheld bool, note string, now time.Time) error {
	if c.Status != StatusOpen {
		return kernel.Invalidf("contest: already resolved (%s)", c.Status)
	}
	if upheld {
		c.Status = StatusUpheld
	} else {
		c.Status = StatusDismissed
	}
	c.Resolution = strings.TrimSpace(note)
	c.ResolvedAt = now
	return nil
}
