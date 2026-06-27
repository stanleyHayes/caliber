package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// TestFlowBEndToEnd is the Flow B (centrepiece) acceptance test (CAL-068): an
// adaptive screening interview that asks one rubric competency at a time, then
// produces a report card with a per-competency score + evidence, an overall
// verdict and confidence, and advances the candidate's Talent Passport to
// screened. It drives the real interview use-case over the in-memory stack +
// deterministic dev model (the streaming transport is tested separately).
func TestFlowBEndToEnd(t *testing.T) {
	ctx := context.Background()
	roleRepo := memory.NewRoleRepo()
	profRepo := memory.NewTalentProfileRepo()
	interviewRepo := memory.NewInterviewRepo()

	candidateID := kernel.NewID()
	profile, err := talent.NewTalentProfile(candidateID, "Backend engineer", []talent.ProfileCompetency{
		{Name: "Go", Level: 4, EvidenceQuote: "built services", SourceSpan: "CV"},
	})
	require.NoError(t, err)
	require.Equal(t, talent.PassportCVOnly, profile.PassportStatus, "starts un-screened")
	require.NoError(t, profRepo.Create(ctx, profile))

	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	require.NoError(t, roleRepo.Create(ctx, rl))

	const maxTurns = 2
	interviewer := interviewapp.NewInterviewer(roleRepo, interviewRepo, llm.NewDev(), maxTurns,
		interviewapp.WithPassportUpdater(profRepo))

	// Start: the first adaptive question is asked.
	iv, q1, err := interviewer.Start(ctx, rl.ID, candidateID, interviewdom.ModeText)
	require.NoError(t, err)
	require.NotNil(t, q1)
	assert.NotEmpty(t, q1.Text)
	assert.NotEmpty(t, q1.CompetencyTag, "the question targets a rubric competency")

	// Answer 1 -> a second, different question (adaptive, not scripted).
	q2, report, err := interviewer.Answer(ctx, iv.ID, "I led a Go service that cut p99 latency by 40%.")
	require.NoError(t, err)
	require.Nil(t, report, "not finished after one answer")
	require.NotNil(t, q2)
	assert.NotEqual(t, q1.CompetencyTag, q2.CompetencyTag, "adaptive: the interview moves across competencies")

	// Answer 2 -> reaches maxTurns and produces the report card.
	_, report, err = interviewer.Answer(ctx, iv.ID, "I designed the SQL schema and indexes for that service.")
	require.NoError(t, err)
	require.NotNil(t, report, "the report card is produced after the final turn")

	// Per-competency scores, each with evidence.
	require.NotEmpty(t, report.Scores)
	for _, sc := range report.Scores {
		assert.NotEmpty(t, sc.Competency, "score names its competency")
		assert.NotEmpty(t, sc.Evidence, "score carries supporting evidence")
		assert.GreaterOrEqual(t, sc.Score, 0.0)
		assert.LessOrEqual(t, sc.Score, 5.0)
	}
	// Overall verdict + confidence are set.
	assert.NotEqual(t, interviewdom.VerdictUnspecified, report.Verdict, "an overall verdict is reached")
	assert.NotEqual(t, kernel.ConfidenceUnknown, report.Confidence, "a confidence is reported")
	assert.NotEmpty(t, report.RecommendedNextStep)

	// The Talent Passport is advanced to screened (Flow B side effect).
	updated, err := profRepo.ByCandidateID(ctx, candidateID)
	require.NoError(t, err)
	assert.Equal(t, talent.PassportScreened, updated.PassportStatus, "the screening advanced the passport")
}
