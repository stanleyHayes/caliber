package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestFlowCEndToEnd is the Flow C demo-beat acceptance test (CAL-075): a verified
// profile, a "run overnight" time-advance that yields a tailored application and
// surfaces a completed screening, and — the hard invariant — no application
// content untraceable to the verified profile. It runs the real agent use-case
// through the gRPC handler over the in-memory stack + deterministic dev model.
func TestFlowCEndToEnd(t *testing.T) {
	ctx := context.Background()
	candRepo := memory.NewCandidateRepo()
	profRepo := memory.NewTalentProfileRepo()
	roleRepo := memory.NewRoleRepo()
	appsRepo := memory.NewApplicationRepo()
	interviewRepo := memory.NewInterviewRepo()

	// A verified candidate in Accra whose profile evidences the role's must-have.
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{Location: "Accra"})
	require.NoError(t, err)
	require.NoError(t, candRepo.Create(ctx, cand))
	profile, err := talent.NewTalentProfile(cand.ID, "Senior Go engineer", []talent.ProfileCompetency{
		{Name: "Go", Level: 5, EvidenceQuote: "led a payments platform in Go", SourceSpan: "CV"},
		{Name: "SQL", Level: 4, EvidenceQuote: "designed Postgres schemas", SourceSpan: "CV"},
	})
	require.NoError(t, err)
	require.NoError(t, profRepo.Create(ctx, profile))

	// An open role the candidate genuinely qualifies for (must-have Go, in Accra).
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid, MustHaves: []string{"Go"}},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1700000000, 0))
	require.NoError(t, err)
	require.NoError(t, rl.Open())
	require.NoError(t, roleRepo.Create(ctx, rl))

	// A previously completed screening for this candidate (carries a report card).
	require.NoError(t, interviewRepo.Create(ctx, &interviewdom.Interview{
		ID:          kernel.NewID(),
		CandidateID: cand.ID,
		RoleID:      rl.ID,
		Report:      &interviewdom.ReportCard{CandidateID: cand.ID, RoleID: rl.ID, Verdict: interviewdom.VerdictAdvance},
	}))

	runner := candidateagentapp.NewAgentRunner(candRepo, profRepo, roleRepo, appsRepo, llm.NewDev(),
		candidateagentapp.WithWakeUpInsights(interviewRepo, memory.NewMatchRepo()))
	srv := NewAgentServer(runner, appsRepo)

	// "Run overnight".
	resp, err := srv.TimeAdvance(asCandidate(ctx, cand.ID), &caliberv1.TimeAdvanceRequest{CandidateId: cand.ID.String()})
	require.NoError(t, err)
	wake := resp.GetWakeUp()
	assert.GreaterOrEqual(t, wake.GetApplicationsSubmitted(), int32(1), "the agent tailored and submitted an application")
	assert.GreaterOrEqual(t, wake.GetScreeningsCompleted(), int32(1), "the completed screening is surfaced")

	// No-fabrication: every submitted application traces to the verified profile,
	// and its tailored summary is grounded by the SAME invariant the agent enforces
	// before submitting (CheckGrounding) — it asserts no skill the profile lacks.
	apps, total, err := appsRepo.ByCandidate(ctx, cand.ID, kernel.NewPage(1, 10))
	require.NoError(t, err)
	require.Positive(t, total)
	profileNames := []string{"Go", "SQL"}
	roleNames := []string{"Go", "SQL"}
	for _, ap := range apps {
		assert.Equal(t, agentdom.SourceAgent, ap.Source, "authored by the agent")
		assert.Equal(t, profile.ID, ap.ProfileID, "grounded in the verified profile")
		assert.Equal(t, agentdom.StatusSubmitted, ap.Status)
		require.NotEmpty(t, ap.TailoredSummary)
		grounding := agentdom.CheckGrounding(ap.TailoredSummary, profileNames, roleNames)
		assert.Truef(t, grounding.Grounded,
			"application content traces to the verified profile; fabricated: %v", grounding.Fabricated)
	}
}
