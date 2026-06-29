package privacy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/app/privacy"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// recorder captures the order in which the cascade touched each port, so a test
// can assert dependents are erased before the aggregate, etc.
type recorder struct {
	order *[]string
	name  string
	err   error
}

func (r recorder) DeleteByCandidate(_ context.Context, _ kernel.ID) error {
	*r.order = append(*r.order, r.name)
	return r.err
}
func (r recorder) Delete(_ context.Context, _ kernel.ID) error {
	*r.order = append(*r.order, r.name)
	return r.err
}
func (r recorder) Anonymize(_ context.Context, _ kernel.ID) error {
	*r.order = append(*r.order, r.name)
	return r.err
}
func (r recorder) TombstoneActor(_ context.Context, _ kernel.ID) error {
	*r.order = append(*r.order, r.name)
	return r.err
}

func TestEraseCandidate_RunsFullCascadeInOrder(t *testing.T) {
	var order []string
	rec := func(name string) recorder { return recorder{order: &order, name: name} }

	eraser := privacy.NewEraser(
		rec("candidate"), rec("identity"), rec("audit"),
		rec("profile"), rec("applications"), rec("interviews"), rec("matches"), rec("contests"),
	)
	require.NoError(t, eraser.EraseCandidate(context.Background(), kernel.NewID()))

	// Scoped records first (in injected order), then the candidate aggregate, then
	// the owning user account, then the audit tombstone.
	assert.Equal(t,
		[]string{"profile", "applications", "interviews", "matches", "contests", "candidate", "identity", "audit"},
		order,
	)
}

func TestEraseCandidate_StopsAndSurfacesScopedFailure(t *testing.T) {
	var order []string
	rec := func(name string) recorder { return recorder{order: &order, name: name} }
	boom := recorder{order: &order, name: "interviews", err: errors.New("db down")}

	eraser := privacy.NewEraser(
		rec("candidate"), rec("identity"), rec("audit"),
		rec("profile"), rec("applications"), boom, rec("matches"),
	)
	err := eraser.EraseCandidate(context.Background(), kernel.NewID())
	require.Error(t, err)
	// It stopped at the failing step — later scoped erasers + the aggregate, user,
	// and audit steps did not run.
	assert.Equal(t, []string{"profile", "applications", "interviews"}, order)
}

func TestEraseCandidate_SurfacesAuditTombstoneFailure(t *testing.T) {
	var order []string
	rec := func(name string) recorder { return recorder{order: &order, name: name} }
	audit := recorder{order: &order, name: "audit", err: errors.New("audit store down")}

	eraser := privacy.NewEraser(rec("candidate"), rec("identity"), audit, rec("profile"))
	err := eraser.EraseCandidate(context.Background(), kernel.NewID())
	require.Error(t, err)
	assert.Equal(t, []string{"profile", "candidate", "identity", "audit"}, order)
}

func TestEraseCandidate_RejectsZeroID(t *testing.T) {
	var order []string
	rec := func(name string) recorder { return recorder{order: &order, name: name} }
	eraser := privacy.NewEraser(rec("candidate"), rec("identity"), rec("audit"), rec("profile"))

	err := eraser.EraseCandidate(context.Background(), kernel.ID(""))
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	assert.Empty(t, order, "nothing is touched for an invalid request")
}
