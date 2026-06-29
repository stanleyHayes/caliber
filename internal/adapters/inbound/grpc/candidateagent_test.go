package grpcadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

const agentAssessJSON = `{"fit_score":0.85,"apply":true,"tailored_summary":"Drawing on verified Go experience, a strong fit."}`

func TestAgentTimeAdvanceThenWakeUpAndList(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	apps := memory.NewApplicationRepo() // real store: ListApplications reads what TimeAdvance writes

	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cid, "s", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1, 0))
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil).AnyTimes()
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil).AnyTimes()
	roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil).AnyTimes()
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: agentAssessJSON}, nil).AnyTimes()

	srv := NewAgentServer(candidateagentapp.NewAgentRunner(candidates, profiles, roles, apps, llm), apps, nil)

	adv, err := srv.TimeAdvance(asCandidate(context.Background(), cid), &caliberv1.TimeAdvanceRequest{CandidateId: cid.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), adv.GetWakeUp().GetApplicationsSubmitted())
	assert.NotEmpty(t, adv.GetWakeUp().GetHighlights())

	wv, err := srv.GetWakeUpView(asCandidate(context.Background(), cid), &caliberv1.GetWakeUpViewRequest{CandidateId: cid.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), wv.GetWakeUp().GetApplicationsSubmitted(), "the last run is remembered")

	la, err := srv.ListApplications(asCandidate(context.Background(), cid),
		&caliberv1.ListApplicationsRequest{CandidateId: cid.String(), Page: &caliberv1.PageRequest{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	require.Len(t, la.GetApplications(), 1)
	got := la.GetApplications()[0]
	assert.Equal(t, caliberv1.ApplicationSource_APPLICATION_SOURCE_AGENT, got.GetSource())
	assert.Equal(t, caliberv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED, got.GetStatus())
	assert.NotEmpty(t, got.GetTailoredSummary())
	assert.Equal(t, int64(1), la.GetPage().GetTotalItems())
}

func TestAgentRunAgentRemembersWakeUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	roles := mocks.NewMockRoleRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	apps := memory.NewApplicationRepo()

	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	profile, err := talent.NewTalentProfile(cid, "s", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	rl, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Location: "Accra", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1, 0))
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil).AnyTimes()
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil).AnyTimes()
	roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil).AnyTimes()
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: agentAssessJSON}, nil).AnyTimes()

	srv := NewAgentServer(candidateagentapp.NewAgentRunner(candidates, profiles, roles, apps, llm), apps, nil)

	// RunAgent completes inline when no dispatcher is configured.
	run, err := srv.RunAgent(asCandidate(context.Background(), cid), &caliberv1.RunAgentRequest{CandidateId: cid.String()})
	require.NoError(t, err)
	assert.Empty(t, run.GetJobId())

	// The live wake-up view reflects the application it submitted.
	wv, err := srv.GetWakeUpView(asCandidate(context.Background(), cid), &caliberv1.GetWakeUpViewRequest{CandidateId: cid.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), wv.GetWakeUp().GetApplicationsSubmitted(), "RunAgent updates the live wake-up view")
}

func TestAgentRunAgentRequiresSelfCandidate(t *testing.T) {
	srv := NewAgentServer(nil, nil, nil)
	target := kernel.NewID()
	// A different candidate cannot run someone else's agent.
	_, err := srv.RunAgent(asCandidate(context.Background(), kernel.NewID()), &caliberv1.RunAgentRequest{CandidateId: target.String()})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	// Unauthenticated is rejected.
	_, err = srv.RunAgent(context.Background(), &caliberv1.RunAgentRequest{CandidateId: target.String()})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAgentGetWakeUpViewEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	srv := NewAgentServer(
		candidateagentapp.NewAgentRunner(
			candidates, profiles,
			mocks.NewMockRoleRepository(ctrl), memory.NewApplicationRepo(), mocks.NewMockLLMClient(ctrl)),
		memory.NewApplicationRepo(),
		nil)
	resp, err := srv.GetWakeUpView(asCandidate(context.Background(), cid), &caliberv1.GetWakeUpViewRequest{CandidateId: cid.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.GetWakeUp().GetApplicationsSubmitted())
}

func TestAgentRunAgentDispatchesWhenQueueConfigured(t *testing.T) {
	cid := kernel.NewID()
	dispatcher := &fakeTaskDispatcher{taskID: "task-123"}
	srv := NewAgentServer(nil, nil, dispatcher)

	resp, err := srv.RunAgent(asCandidate(context.Background(), cid), &caliberv1.RunAgentRequest{CandidateId: cid.String()})

	require.NoError(t, err)
	assert.Equal(t, "task-123", resp.GetJobId())
	assert.Equal(t, cid, dispatcher.candidateID)
}

func asCandidate(ctx context.Context, candidateID kernel.ID) context.Context {
	return context.WithValue(ctx, principalKey{}, app.Principal{UserID: candidateID, Role: identity.RoleCandidate.String()})
}

// asEmployer builds a context whose principal IS the given employer (employers are
// users; a role's EmployerID is the owning user's id), so ownership checks pass.
func asEmployer(ctx context.Context, userID kernel.ID) context.Context {
	return context.WithValue(ctx, principalKey{}, app.Principal{UserID: userID, Role: identity.RoleEmployer.String()})
}

func TestAgentRequiresSelfCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv := NewAgentServer(
		candidateagentapp.NewAgentRunner(
			mocks.NewMockCandidateRepository(ctrl), mocks.NewMockTalentProfileRepository(ctrl),
			mocks.NewMockRoleRepository(ctrl), memory.NewApplicationRepo(), mocks.NewMockLLMClient(ctrl)),
		memory.NewApplicationRepo(),
		nil)
	target := kernel.NewID()

	// A different candidate cannot run someone else's agent.
	_, err := srv.TimeAdvance(asCandidate(context.Background(), kernel.NewID()),
		&caliberv1.TimeAdvanceRequest{CandidateId: target.String()})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// An unauthenticated caller is rejected.
	_, err = srv.TimeAdvance(context.Background(), &caliberv1.TimeAdvanceRequest{CandidateId: target.String()})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

type fakeTaskDispatcher struct {
	taskID      string
	candidateID kernel.ID
}

func (f *fakeTaskDispatcher) DispatchCandidateAgentRun(
	_ context.Context,
	candidateID kernel.ID,
	_ ...appqueue.DispatchOption,
) (string, error) {
	f.candidateID = candidateID
	return f.taskID, nil
}

func (f *fakeTaskDispatcher) DispatchInterviewScoring(
	context.Context,
	kernel.ID,
	...appqueue.DispatchOption,
) (string, error) {
	return "", nil
}

func (f *fakeTaskDispatcher) DispatchBatchRematch(
	context.Context,
	kernel.ID,
	...appqueue.DispatchOption,
) (string, error) {
	return "", nil
}

func (f *fakeTaskDispatcher) Close() error { return nil }
