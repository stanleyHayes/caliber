package privacy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// RetentionEraser erases a single candidate's full record. *Eraser satisfies it,
// so the retention sweep reuses the exact right-to-erasure cascade (CAL-118).
type RetentionEraser interface {
	EraseCandidate(ctx context.Context, candidateID kernel.ID) error
}

// RetentionLister finds candidates whose personal data has aged past the
// retention window — those whose account was created on or before the cutoff.
type RetentionLister interface {
	ListEligibleForRetention(ctx context.Context, createdOnOrBefore time.Time) ([]kernel.ID, error)
}

// RetentionResult summarizes one sweep for logging/audit.
type RetentionResult struct {
	Eligible int // candidates found past the retention window
	Erased   int // successfully erased
	Failed   int // erasure attempts that errored (left for the next run to retry)
}

// RetentionSweeper enforces the data-retention schedule (CAL-158), operationalizing
// the right-to-erasure cascade (CAL-118): it finds candidate records older than the
// configured window and erases each, so personal data is not retained past its
// lawful basis (Ghana DPA 2012). Every erasure is audited by the underlying Eraser
// (a retained-but-tombstoned audit entry), and the sweep summary is returned so the
// job can log how much was purged.
type RetentionSweeper struct {
	lister    RetentionLister
	eraser    RetentionEraser
	retention time.Duration
	now       func() time.Time
}

// NewRetentionSweeper wires the sweep. A non-positive retention window disables
// it (Sweep becomes a no-op), so retention is off unless explicitly configured.
// now defaults to time.Now.
func NewRetentionSweeper(lister RetentionLister, eraser RetentionEraser, retention time.Duration, now func() time.Time) *RetentionSweeper {
	if now == nil {
		now = time.Now
	}
	return &RetentionSweeper{lister: lister, eraser: eraser, retention: retention, now: now}
}

// Sweep erases every candidate whose data has aged past the retention window. A
// disabled (non-positive) window is a no-op — the lister is never queried. One
// candidate's failure does not stop the sweep: the rest still run, and the sweep
// returns an aggregate error so the failure is visible and the job retries (the
// erasure is effectively idempotent — an already-erased candidate no longer
// appears eligible, so a retry only revisits the ones that failed).
func (s *RetentionSweeper) Sweep(ctx context.Context) (RetentionResult, error) {
	if s.retention <= 0 {
		return RetentionResult{}, nil
	}
	cutoff := s.now().Add(-s.retention)
	ids, err := s.lister.ListEligibleForRetention(ctx, cutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("privacy: list retention-eligible: %w", err)
	}
	result := RetentionResult{Eligible: len(ids)}
	var errs []error
	for _, id := range ids {
		if eerr := s.eraser.EraseCandidate(ctx, id); eerr != nil {
			result.Failed++
			errs = append(errs, fmt.Errorf("erase %s: %w", id, eerr))
			continue
		}
		result.Erased++
	}
	if len(errs) > 0 {
		return result, fmt.Errorf("privacy: retention sweep had %d failure(s): %w", len(errs), errors.Join(errs...))
	}
	return result, nil
}
