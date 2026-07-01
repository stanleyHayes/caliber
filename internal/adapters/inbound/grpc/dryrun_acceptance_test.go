package grpcadapter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/embeddings"
	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	dashboardapp "github.com/xcreativs/caliber/internal/app/dashboard"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/app/roles"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TestFullDryRunAcceptanceSweep is CAL-108: one rehearsal that drives the entire
// demo narrative on realistic seed data and asserts every §15 acceptance
// criterion passes. The path is Frame → Flow A → Flow B → Flow C → Radar close,
// with a trust gate at the end.
func TestFullDryRunAcceptanceSweep(t *testing.T) {
	ctx := context.Background()

	// ----------------------------------------------------------------------
	// Shared in-memory infrastructure + deterministic dev AI.
	// ----------------------------------------------------------------------
	users := memory.NewUserRepo()
	candRepo := memory.NewCandidateRepo()
	profRepo := memory.NewTalentProfileRepo()
	roleRepo := memory.NewRoleRepo()
	interviewRepo := memory.NewInterviewRepo()
	appsRepo := memory.NewApplicationRepo()
	matchRepo := memory.NewMatchRepo()
	auditRepo := memory.NewAuditRepo()

	llmDev := llm.NewDev()
	embedDev := embeddings.NewDev()

	// ----------------------------------------------------------------------
	// Seed identities and a realistic Ghana-context pool.
	// ----------------------------------------------------------------------
	empUser := seedUser(ctx, t, users, "Employer One", identity.RoleEmployer)
	kofiUser := seedUser(ctx, t, users, "Kofi Mensah", identity.RoleCandidate)
	amaUser := seedUser(ctx, t, users, "Ama Mensah", identity.RoleCandidate)
	esiUser := seedUser(ctx, t, users, "Esi Mensah", identity.RoleCandidate)

	kofi := seedCandidate(ctx, t, candRepo, profRepo, kofiUser.ID, "Accra", []talent.ProfileCompetency{
		{Name: "Core skills", Level: 5, EvidenceQuote: "built production services", SourceSpan: "CV"},
		{Name: "Communication", Level: 4, EvidenceQuote: "led daily stand-ups", SourceSpan: "CV"},
		{Name: "System design", Level: 4, EvidenceQuote: "designed distributed systems", SourceSpan: "CV"},
	})
	ama := seedCandidate(ctx, t, candRepo, profRepo, amaUser.ID, "Accra", []talent.ProfileCompetency{
		{Name: "React Native", Level: 5, EvidenceQuote: "shipped iOS and Android apps", SourceSpan: "CV"},
		{Name: "Mobile", Level: 4, EvidenceQuote: "led mobile team", SourceSpan: "CV"},
	})
	esi := seedCandidate(ctx, t, candRepo, profRepo, esiUser.ID, "Accra", []talent.ProfileCompetency{
		{Name: "Communication", Level: 5, EvidenceQuote: "client presentations", SourceSpan: "CV"},
		{Name: "System design", Level: 4, EvidenceQuote: "designed APIs", SourceSpan: "CV"},
	})

	// Pre-existing open role and completed screening for Flow C's wake-up view.
	mobileRole := seedOpenRole(ctx, t, roleRepo, empUser.ID, "Mobile Engineer", "Accra", role.SeniorityMid,
		[]string{"React Native"},
		role.Rubric{Competencies: []role.Competency{
			{Name: "React Native", Weight: 0.6, MustHave: true},
			{Name: "Mobile", Weight: 0.4},
		}})
	seedCompletedInterview(ctx, t, interviewRepo, ama.ID, mobileRole.ID)

	// ----------------------------------------------------------------------
	// Wire the full hexagonal stack behind the gRPC adapters.
	// ----------------------------------------------------------------------
	recaller := memory.NewRecaller(candRepo)
	shortlister := matchingapp.NewShortlister(roleRepo, candRepo, profRepo, recaller, embedDev, llmDev, matchRepo)
	refiner := matchingapp.NewRefiner(roleRepo, shortlister)
	rejections := matchingapp.NewRejectionRecorder(roleRepo, auditRepo, time.Now)

	roleSrv := NewRoleServer(
		roles.NewSpecGenerator(llmDev, roleRepo, time.Now),
		roles.NewSpecEditor(roleRepo),
		shortlister,
	)
	matchSrv := NewMatchServer(shortlister, refiner, rejections)
	interviewer := interviewapp.NewInterviewer(roleRepo, interviewRepo, llmDev,
		interviewdom.Config{MaxQuestions: 2},
		interviewapp.WithPassportUpdater(profRepo))
	interviewSrv := NewInterviewServer(interviewer)
	agentRunner := candidateagentapp.NewAgentRunner(candRepo, profRepo, roleRepo, appsRepo, llmDev,
		candidateagentapp.WithWakeUpInsights(interviewRepo, matchRepo),
		candidateagentapp.WithAuditTrail(auditRepo, time.Now))
	agentSrv := NewAgentServer(agentRunner, appsRepo, nil)
	dashboardSrv := NewDashboardServer(dashboardapp.NewAggregator(candRepo, profRepo, users, roleRepo))
	auditSrv := NewAuditServer(auditRepo)

	employerCtx := asEmployer(ctx, empUser.ID)
	kofiCtx := asCandidate(ctx, kofi.ID)
	amaCtx := asCandidate(ctx, ama.ID)
	reviewerCtx := asRole(ctx, identity.RoleEmployer)

	page := &caliberv1.PageRequest{Page: 1, PageSize: 10}

	// ===================================================================
	// FRAME — Talent Radar baseline (§15.4)
	// ===================================================================
	pool, err := dashboardSrv.GetPool(reviewerCtx, &caliberv1.GetPoolRequest{Page: page})
	require.NoError(t, err, "Radar pool loads")
	require.Len(t, pool.GetCandidates(), 3, "live pool shows all seeded candidates")

	sd, err := dashboardSrv.GetSupplyDemand(reviewerCtx, &caliberv1.GetSupplyDemandRequest{})
	require.NoError(t, err, "Radar supply/demand loads")
	require.NotEmpty(t, sd.GetItems(), "supply/demand snapshot is populated")

	alerts, err := dashboardSrv.GetAlerts(reviewerCtx, &caliberv1.GetAlertsRequest{Page: page})
	require.NoError(t, err, "Radar alerts load")
	require.NotEmpty(t, alerts.GetAlerts(), "two-way match alerts are raised on seed data")

	baselineTTL, err := dashboardSrv.GetTimeToShortlist(reviewerCtx, &caliberv1.GetTimeToShortlistRequest{})
	require.NoError(t, err, "time-to-shortlist headline loads")
	require.NotNil(t, baselineTTL.GetMetric())
	assert.Greater(t, baselineTTL.GetMetric().GetBaselineHours(), baselineTTL.GetMetric().GetCurrentHours(),
		"weeks-to-hours collapse is visible")

	// ===================================================================
	// FLOW A — Employer intake & explainable shortlisting (§15.1)
	// ===================================================================
	genResp, err := roleSrv.GenerateRoleSpec(employerCtx, &caliberv1.GenerateRoleSpecRequest{
		EmployerId: empUser.ID.String(),
		FreeText:   "Senior Go engineer in Accra to lead our payments platform",
	})
	require.NoError(t, err, "messy sentence produces a structured role spec")
	generatedRole := genResp.GetRole()
	require.NotNil(t, generatedRole)
	assert.NotEmpty(t, generatedRole.GetTitle(), "spec has a title")
	require.NotEmpty(t, generatedRole.GetRubric().GetCompetencies(), "spec has a weighted rubric")
	assert.GreaterOrEqual(t, genResp.GetAvailableMatches(), int32(1),
		"instant availability signal is reported")

	backendRoleID := generatedRole.GetId()

	// Ranked, explainable shortlist.
	slResp, err := matchSrv.GenerateShortlist(employerCtx, &caliberv1.GenerateShortlistRequest{
		RoleId: backendRoleID,
		Page:   page,
	})
	require.NoError(t, err, "shortlist generates")
	sl := slResp.GetShortlist()
	matches := sl.GetMatches()
	require.NotEmpty(t, matches, "at least one candidate makes the shortlist")

	// Ranked descending by fit.
	for i := 1; i < len(matches); i++ {
		assert.GreaterOrEqual(t, matches[i-1].GetOverallScore(), matches[i].GetOverallScore(),
			"shortlist is ranked by overall fit")
	}

	// Explainability contract: no black-box fields.
	top := matches[0]
	assert.Positive(t, top.GetOverallScore())
	assert.NotEqual(t, caliberv1.Confidence_CONFIDENCE_UNSPECIFIED, top.GetConfidence())
	assert.NotEmpty(t, top.GetRationale(), "plain-English rationale is present")
	require.NotEmpty(t, top.GetBreakdown(), "per-competency breakdown is present")
	for _, b := range top.GetBreakdown() {
		assert.NotEmpty(t, b.GetCompetency())
		assert.NotEmpty(t, b.GetEvidence(), "every score cites evidence")
	}
	assert.NotNil(t, sl.GetPage(), "shortlist is paginated")

	// Exclusions are surfaced, never silent.
	assert.NotEmpty(t, sl.GetExclusions(), "candidates gated out are explained")
	assert.GreaterOrEqual(t, sl.GetPoolDepth(), int32(1), "pool depth reflects strong matches")

	// Refine: change the rubric so a previously excluded candidate becomes the top match.
	refinedResp, err := matchSrv.RefineShortlist(employerCtx, &caliberv1.RefineShortlistRequest{
		RoleId: backendRoleID,
		Spec:   generatedRole.GetSpec(),
		Rubric: &caliberv1.Rubric{
			Competencies: []*caliberv1.Competency{
				{Name: "Communication", Weight: 0.5, MustHave: true},
				{Name: "System design", Weight: 0.5, MustHave: false},
			},
		},
		Page: page,
	})
	require.NoError(t, err, "refine re-ranks the shortlist")
	refined := refinedResp.GetShortlist()
	require.NotEmpty(t, refined.GetMatches(), "refined shortlist still has matches")
	assert.Equal(t, esi.ID.String(), refined.GetMatches()[0].GetCandidateId(),
		"refining criteria re-ranks: Esi (Communication 5) now tops Kofi (Communication 4)")

	// ===================================================================
	// FLOW B — AI screening interview (§15.2)
	// ===================================================================
	streamCtx, cancel := context.WithCancel(kofiCtx)
	defer cancel()
	stream := &fakeInterviewStream{ctx: streamCtx}
	startDone := make(chan error, 1)
	go func() {
		startDone <- interviewSrv.StartInterview(&caliberv1.StartInterviewRequest{
			RoleId:      backendRoleID,
			CandidateId: kofi.ID.String(),
			Mode:        caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
		}, stream)
	}()

	require.Eventually(t, func() bool { return len(stream.messages()) >= 2 }, 2*time.Second, 5*time.Millisecond,
		"interview opens and asks the first question")
	msgs := stream.messages()
	require.Equal(t, "open", msgs[0].GetStatus().GetState())
	q1 := msgs[1].GetQuestion()
	require.NotNil(t, q1)
	assert.NotEmpty(t, q1.GetText())
	assert.NotEmpty(t, q1.GetCompetencyTag(), "first question targets a rubric competency")

	// Answer 1 -> adaptive second question (different competency).
	_, err = interviewSrv.SubmitAnswer(kofiCtx, &caliberv1.SubmitAnswerRequest{
		InterviewId: q1.GetInterviewId(),
		Answer:      "I led a Go service that cut p99 latency by 40%.",
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		for _, m := range stream.messages() {
			if q := m.GetQuestion(); q != nil && q.GetOrdinal() != q1.GetOrdinal() {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond, "adaptive second question arrives")

	var q2 *caliberv1.InterviewQuestion
	for _, m := range stream.messages() {
		if q := m.GetQuestion(); q != nil && q.GetInterviewId() == q1.GetInterviewId() && q.GetOrdinal() != q1.GetOrdinal() {
			q2 = q
		}
	}
	require.NotNil(t, q2)
	assert.NotEqual(t, q1.GetCompetencyTag(), q2.GetCompetencyTag(),
		"interview adapts across competencies, not a fixed script")

	// Answer 2 -> report card streamed.
	_, err = interviewSrv.SubmitAnswer(kofiCtx, &caliberv1.SubmitAnswerRequest{
		InterviewId: q2.GetInterviewId(),
		Answer:      "I designed the SQL schema and indexes for that service.",
	})
	require.NoError(t, err)
	select {
	case err := <-startDone:
		require.NoError(t, err, "stream closes cleanly after the report card")
	case <-time.After(2 * time.Second):
		t.Fatal("interview did not finish after final answer")
	}

	var card *caliberv1.ReportCard
	for _, m := range stream.messages() {
		if m.GetReportCard() != nil {
			card = m.GetReportCard()
		}
	}
	require.NotNil(t, card, "report card is streamed")
	assert.NotEqual(t, caliberv1.InterviewVerdict_INTERVIEW_VERDICT_UNSPECIFIED, card.GetVerdict(),
		"overall verdict is set")
	assert.NotEqual(t, caliberv1.Confidence_CONFIDENCE_UNSPECIFIED, card.GetConfidence(),
		"confidence is set")
	assert.NotEmpty(t, card.GetRecommendedNextStep())
	require.NotEmpty(t, card.GetScores(), "per-competency scores are present")
	for _, sc := range card.GetScores() {
		assert.NotEmpty(t, sc.GetCompetency())
		assert.NotEmpty(t, sc.GetEvidence(), "every interview score cites transcript evidence")
	}

	// Report card is also retrievable via unary RPC.
	reportResp, err := interviewSrv.GetReportCard(employerCtx, &caliberv1.GetReportCardRequest{
		InterviewId: q1.GetInterviewId(),
	})
	require.NoError(t, err, "stored report card is viewable by the role owner")
	require.NotNil(t, reportResp.GetReportCard())

	// Talent Passport advanced to screened.
	kofiProfileAfter, err := profRepo.ByCandidateID(ctx, kofi.ID)
	require.NoError(t, err)
	assert.Equal(t, talent.PassportScreened, kofiProfileAfter.PassportStatus,
		"Flow B advances the Talent Passport")

	// ===================================================================
	// FLOW C — Candidate agent "works while you sleep, honestly" (§15.3)
	// ===================================================================
	flowCResp, err := agentSrv.TimeAdvance(amaCtx, &caliberv1.TimeAdvanceRequest{CandidateId: ama.ID.String()})
	require.NoError(t, err, "time-advance runs")
	wake := flowCResp.GetWakeUp()
	assert.GreaterOrEqual(t, wake.GetApplicationsSubmitted(), int32(1),
		"agent tailored and submitted an application")
	assert.GreaterOrEqual(t, wake.GetScreeningsCompleted(), int32(1),
		"completed screening is surfaced in the wake-up view")

	// No fabrication: every submitted application traces to the verified profile.
	apps, total, err := appsRepo.ByCandidate(ctx, ama.ID, kernel.NewPage(1, 10))
	require.NoError(t, err)
	require.Positive(t, total)
	profileNames := []string{"React Native", "Mobile"}
	roleNames := []string{"React Native", "Mobile"}
	for _, ap := range apps {
		assert.Equal(t, agentdom.SourceAgent, ap.Source, "application is authored by the agent")
		assert.Equal(t, amaProfileID(t, ama.ID, profRepo), ap.ProfileID, "application is grounded in the verified profile")
		assert.Equal(t, agentdom.StatusSubmitted, ap.Status)
		require.NotEmpty(t, ap.TailoredSummary)
		grounding := agentdom.CheckGrounding(ap.TailoredSummary, profileNames, roleNames)
		assert.Truef(t, grounding.Grounded,
			"application content traces to the verified profile; fabricated: %v", grounding.Fabricated)
	}

	// Agent action is recorded in the audit trail.
	for _, ap := range apps {
		appAudit, err := auditSrv.ListAuditLog(employerCtx, &caliberv1.ListAuditLogRequest{
			Entity:   "application",
			EntityId: ap.ID.String(),
			Page:     page,
		})
		require.NoError(t, err)
		require.NotEmpty(t, appAudit.GetEntries(), "each agent application is audited")
		assert.Equal(t, "agent_submit", appAudit.GetEntries()[0].GetAction(),
			"the audit log records the autonomous submission")
	}

	// ===================================================================
	// CLOSE — Radar reflects the updated pool (§15.4)
	// ===================================================================
	closePool, err := dashboardSrv.GetPool(reviewerCtx, &caliberv1.GetPoolRequest{Page: page})
	require.NoError(t, err)
	var kofiRow *caliberv1.PoolCandidate
	for _, c := range closePool.GetCandidates() {
		if c.GetCandidateId() == kofi.ID.String() {
			kofiRow = c
			break
		}
	}
	require.NotNil(t, kofiRow, "Kofi remains in the live pool")
	assert.Equal(t, caliberv1.PassportStatus_PASSPORT_STATUS_SCREENED, kofiRow.GetPassportStatus(),
		"pool reflects Kofi's screened passport after Flow B")

	closeAlerts, err := dashboardSrv.GetAlerts(reviewerCtx, &caliberv1.GetAlertsRequest{Page: page})
	require.NoError(t, err)
	assert.NotEmpty(t, closeAlerts.GetAlerts(), "alerts remain after the flows")

	closeTTL, err := dashboardSrv.GetTimeToShortlist(reviewerCtx, &caliberv1.GetTimeToShortlistRequest{})
	require.NoError(t, err)
	assert.Greater(t, closeTTL.GetMetric().GetBaselineHours(), closeTTL.GetMetric().GetCurrentHours(),
		"closing headline still shows weeks → hours")

	// ===================================================================
	// TRUST — Human-approval gate before any rejection (§15.4)
	// ===================================================================
	// A rejection without explicit human approval is rejected by the system.
	_, err = matchSrv.RecordRejection(employerCtx, &caliberv1.RecordRejectionRequest{
		RoleId:        mobileRole.ID.String(),
		CandidateId:   esi.ID.String(),
		Reason:        "Role filled by another candidate",
		HumanApproved: false,
	})
	require.Error(t, err, "the AI never auto-rejects")

	// A human-approved rejection is logged.
	rejResp, err := matchSrv.RecordRejection(employerCtx, &caliberv1.RecordRejectionRequest{
		RoleId:        mobileRole.ID.String(),
		CandidateId:   esi.ID.String(),
		Reason:        "Not the right fit after review",
		HumanApproved: true,
	})
	require.NoError(t, err, "human-approved rejection succeeds")
	require.NotEmpty(t, rejResp.GetAuditEntryId(), "rejection returns the audit entry id")

	rejectionAudit, err := auditSrv.ListAuditLog(employerCtx, &caliberv1.ListAuditLogRequest{
		Entity:   "match",
		EntityId: esi.ID.String(),
		Page:     page,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rejectionAudit.GetEntries(), "human-approved rejection is in the audit trail")
	assert.Equal(t, "approve_rejection", rejectionAudit.GetEntries()[0].GetAction(),
		"the audit log records the human approval")
}

func seedUser(ctx context.Context, t *testing.T, repo *memory.UserRepo, name string, role identity.Role) *identity.User {
	t.Helper()
	email, err := identity.NewEmail(nameToEmail(name))
	require.NoError(t, err)
	u, err := identity.NewUser(email, role, name, "hash", time.Unix(1, 0))
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, u))
	return u
}

func nameToEmail(name string) string {
	// Simple deterministic email for test seeding.
	return strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@example.com"
}

func seedCandidate(
	ctx context.Context, t *testing.T,
	candRepo *memory.CandidateRepo, profRepo *memory.TalentProfileRepo,
	userID kernel.ID, location string, comps []talent.ProfileCompetency,
) *talent.Candidate {
	t.Helper()
	c, err := talent.NewCandidate(userID, location, talent.CandidateIntake{Location: location})
	require.NoError(t, err)
	require.NoError(t, candRepo.Create(ctx, c))
	p, err := talent.NewTalentProfile(c.ID, "verified profile", comps)
	require.NoError(t, err)
	require.NoError(t, profRepo.Create(ctx, p))
	return c
}

func seedOpenRole(
	ctx context.Context, t *testing.T, repo *memory.RoleRepo,
	employerID kernel.ID, title, location string, seniority role.Seniority,
	mustHaves []string, rubric role.Rubric,
) *role.Role {
	t.Helper()
	rl, err := role.NewRole(employerID, role.RoleSpec{
		Title:        title,
		Location:     location,
		Seniority:    seniority,
		Availability: "within 1 month",
		MustHaves:    mustHaves,
	}, rubric, time.Unix(1700000000, 0))
	require.NoError(t, err)
	require.NoError(t, rl.Open())
	require.NoError(t, repo.Create(ctx, rl))
	return rl
}

func seedCompletedInterview(
	ctx context.Context, t *testing.T, repo *memory.InterviewRepo,
	candidateID, roleID kernel.ID,
) {
	t.Helper()
	require.NoError(t, repo.Create(ctx, &interviewdom.Interview{
		ID:          kernel.NewID(),
		CandidateID: candidateID,
		RoleID:      roleID,
		Report: &interviewdom.ReportCard{
			CandidateID:         candidateID,
			RoleID:              roleID,
			Verdict:             interviewdom.VerdictAdvance,
			Confidence:          kernel.ConfidenceMedium,
			RecommendedNextStep: "Proceed to onsite.",
			Scores: []interviewdom.CompetencyScore{
				{Competency: "React Native", Score: 4.5, Evidence: "Shipped cross-platform apps."},
			},
		},
	}))
}

func amaProfileID(t *testing.T, candidateID kernel.ID, profRepo *memory.TalentProfileRepo) kernel.ID {
	t.Helper()
	p, err := profRepo.ByCandidateID(context.Background(), candidateID)
	require.NoError(t, err)
	return p.ID
}
