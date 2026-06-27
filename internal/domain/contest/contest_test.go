package contest_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/domain/contest"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func ts() time.Time { return time.Unix(1700000000, 0) }

func TestNewContest_Valid(t *testing.T) {
	cid, sid := kernel.NewID(), kernel.NewID()
	c, err := contest.NewContest(cid, sid, contest.SubjectMatch, "  the breakdown ignored my recent work  ", ts())
	require.NoError(t, err)
	assert.Equal(t, cid, c.CandidateID)
	assert.Equal(t, sid, c.SubjectID)
	assert.Equal(t, contest.SubjectMatch, c.Subject)
	assert.Equal(t, contest.StatusOpen, c.Status)
	assert.Equal(t, "the breakdown ignored my recent work", c.Reason, "reason is trimmed")
	assert.False(t, c.ID.IsZero())
}

func TestNewContest_Validation(t *testing.T) {
	id := kernel.NewID()
	cases := []struct {
		name     string
		cid, sid kernel.ID
		subject  contest.Subject
		reason   string
	}{
		{"zero candidate", kernel.ID(""), id, contest.SubjectMatch, "r"},
		{"zero subject id", id, kernel.ID(""), contest.SubjectMatch, "r"},
		{"invalid subject", id, id, contest.SubjectUnspecified, "r"},
		{"blank reason", id, id, contest.SubjectMatch, "   "},
		{"reason too long", id, id, contest.SubjectMatch, strings.Repeat("x", contest.MaxReasonLen+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := contest.NewContest(tc.cid, tc.sid, tc.subject, tc.reason, ts())
			assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
		})
	}
}

func TestContest_Resolve(t *testing.T) {
	id := kernel.NewID()
	c, err := contest.NewContest(id, id, contest.SubjectReportCard, "evidence misquoted", ts())
	require.NoError(t, err)

	resolvedAt := ts().Add(time.Hour)
	require.NoError(t, c.Resolve(true, "  agreed; rescoring  ", resolvedAt))
	assert.Equal(t, contest.StatusUpheld, c.Status)
	assert.Equal(t, "agreed; rescoring", c.Resolution)
	assert.Equal(t, resolvedAt, c.ResolvedAt)

	// A second resolution is rejected.
	err = c.Resolve(false, "x", resolvedAt)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestContest_ResolveDismiss(t *testing.T) {
	id := kernel.NewID()
	c, _ := contest.NewContest(id, id, contest.SubjectMatch, "reason", ts())
	require.NoError(t, c.Resolve(false, "reviewed", ts()))
	assert.Equal(t, contest.StatusDismissed, c.Status)
}

func TestSubject_StringAndParse(t *testing.T) {
	for _, s := range []contest.Subject{contest.SubjectMatch, contest.SubjectReportCard} {
		parsed, err := contest.ParseSubject(s.String())
		require.NoError(t, err)
		assert.Equal(t, s, parsed)
	}
	assert.Equal(t, "unspecified", contest.SubjectUnspecified.String())
	_, err := contest.ParseSubject("nonsense")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestStatus_String(t *testing.T) {
	assert.Equal(t, "open", contest.StatusOpen.String())
	assert.Equal(t, "upheld", contest.StatusUpheld.String())
	assert.Equal(t, "dismissed", contest.StatusDismissed.String())
}
