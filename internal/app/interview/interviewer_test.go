package interview_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

const (
	questionJSON = `{"question":"Tell me about a Go service you built.","competency_tag":"Go"}`
	reportJSON   = `{"verdict":"advance","confidence":"high","scores":[{"competency":"Go","score":4.5,"evidence":"built a payments service in Go"}],"recommended_next_step":"Schedule an onsite."}`
)

type deps struct {
	roles      *mocks.MockRoleRepository
	interviews *mocks.MockInterviewRepository
	llm        *mocks.MockLLMClient
}

func newDeps(ctrl *gomock.Controller) deps {
	return deps{
		roles:      mocks.NewMockRoleRepository(ctrl),
		interviews: mocks.NewMockInterviewRepository(ctrl),
		llm:        mocks.NewMockLLMClient(ctrl),
	}
}

func sampleRole(t *testing.T) *role.Role {
	t.Helper()
	r, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 0.6, MustHave: true}, {Name: "SQL", Weight: 0.4}}},
		time.Unix(1, 0))
	require.NoError(t, err)
	return r
}

// askingInterview returns an interview in the Asking state with a pending Q1.
func askingInterview(t *testing.T, roleID kernel.ID) *interviewdom.Interview {
	t.Helper()
	iv, err := interviewdom.NewInterview(roleID, kernel.NewID(), interviewdom.ModeText)
	require.NoError(t, err)
	require.NoError(t, iv.Transition(interviewdom.StateAsking))
	require.NoError(t, iv.Ask("Tell me about your Go experience.", "Go"))
	return iv
}

func TestStartOpensAndAsksFirstQuestion(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: questionJSON}, nil)
	d.interviews.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	iv := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	got, q, err := iv.Start(context.Background(), rl.ID, kernel.NewID(), interviewdom.ModeText)
	require.NoError(t, err)
	assert.Equal(t, interviewdom.StateAsking, got.State)
	require.NotNil(t, q)
	assert.Equal(t, "Tell me about a Go service you built.", q.Text)
	assert.Equal(t, "Go", q.CompetencyTag)
}

func TestAnswerAsksNextQuestion(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: questionJSON}, nil)
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	pending, report, err := interviewer.Answer(context.Background(), iv.ID, "I built a payments service in Go.")
	require.NoError(t, err)
	assert.Nil(t, report)
	require.NotNil(t, pending)
	assert.Len(t, iv.Turns, 1, "the answered question became a turn")
}

func TestAnswerCompletesAndScores(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: reportJSON}, nil)
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// maxTurns = 1 -> the first answer completes the interview.
	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 1)
	pending, report, err := interviewer.Answer(context.Background(), iv.ID, "I built a payments service in Go.")
	require.NoError(t, err)
	assert.Nil(t, pending)
	require.NotNil(t, report)
	assert.Equal(t, interviewdom.VerdictAdvance, report.Verdict)
	assert.Equal(t, kernel.ConfidenceHigh, report.Confidence)
	require.Len(t, report.Scores, 1)
	assert.NotEmpty(t, report.Scores[0].Evidence, "every score cites evidence")
	assert.Equal(t, interviewdom.StateClosed, iv.State)
}

func TestReportNotReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	iv := askingInterview(t, kernel.NewID()) // open, no report
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	_, err := interviewer.Report(context.Background(), iv.ID)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestStartRoleNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	d.roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	_, _, err := interviewer.Start(context.Background(), kernel.NewID(), kernel.NewID(), interviewdom.ModeText)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}

func TestAnswerRejectsBadQuestionJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: "not json"}, nil).Times(app.DefaultLLMAttempts)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	_, _, err := interviewer.Answer(context.Background(), iv.ID, "an answer")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestAnswerRejectsBadReportJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: "not json"}, nil).Times(app.DefaultLLMAttempts)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 1)
	_, _, err := interviewer.Answer(context.Background(), iv.ID, "an answer")
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestAnswerScoresDeclineLowConfidence(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{
		Text: `{"verdict":"decline","confidence":"low","scores":[{"competency":"Go","score":1,"evidence":"vague answer"}],"recommended_next_step":"Pass."}`,
	}, nil)
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 1)
	_, report, err := interviewer.Answer(context.Background(), iv.ID, "um, not sure")
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, interviewdom.VerdictDecline, report.Verdict)
	assert.Equal(t, kernel.ConfidenceLow, report.Confidence)
}

func TestReportReturnsCompletedCard(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	// Drive it to completion via Answer (maxTurns=1), then fetch the report.
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil).Times(2)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: reportJSON}, nil)
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 1)
	_, _, err := interviewer.Answer(context.Background(), iv.ID, "built payments in Go")
	require.NoError(t, err)
	card, err := interviewer.Report(context.Background(), iv.ID)
	require.NoError(t, err)
	assert.Equal(t, interviewdom.VerdictAdvance, card.Verdict)
}

func TestAnswerMarksPassportScreenedOnCompletion(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)

	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: reportJSON}, nil)
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	prof, err := talent.NewTalentProfile(iv.CandidateID, "s", []talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "x"}})
	require.NoError(t, err)
	profiles.EXPECT().ByCandidateID(gomock.Any(), iv.CandidateID).Return(prof, nil)
	var updated *talent.TalentProfile
	profiles.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *talent.TalentProfile) error { updated = p; return nil })

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 1, interviewapp.WithPassportUpdater(profiles))
	_, report, err := interviewer.Answer(context.Background(), iv.ID, "concrete answer")
	require.NoError(t, err)
	require.NotNil(t, report)
	require.NotNil(t, updated)
	assert.Equal(t, talent.PassportScreened, updated.PassportStatus, "passport advanced to screened")
}

func TestAnswerAppliesHonestSignalPressureOnVagueAnswer(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	var captured app.LLMRequest
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
			captured = req
			return app.LLMResponse{Text: questionJSON}, nil
		})
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	_, _, err := interviewer.Answer(context.Background(), iv.ID, "It was good, basically various stuff.")
	require.NoError(t, err)
	assert.Contains(t, captured.Prompt, "presses for a specific, real example",
		"a vague answer must trigger honest-signal pressure on the next question (CAL-063)")
}

func TestAnswerNoPressureOnConcreteAnswer(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	rl := sampleRole(t)
	iv := askingInterview(t, rl.ID)
	d.interviews.EXPECT().ByID(gomock.Any(), iv.ID).Return(iv, nil)
	d.roles.EXPECT().ByID(gomock.Any(), rl.ID).Return(rl, nil)
	var captured app.LLMRequest
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
			captured = req
			return app.LLMResponse{Text: questionJSON}, nil
		})
	d.interviews.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	interviewer := interviewapp.NewInterviewer(d.roles, d.interviews, d.llm, 4)
	_, _, err := interviewer.Answer(context.Background(), iv.ID,
		"I led the migration of our payments service to Go and cut p99 latency by 40%.")
	require.NoError(t, err)
	assert.NotContains(t, captured.Prompt, "presses for a specific, real example",
		"a concrete answer should not trigger the pressure directive")
}
