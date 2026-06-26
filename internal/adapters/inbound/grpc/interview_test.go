package grpcadapter

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/app"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"github.com/xcreativs/caliber/internal/mocks"
)

const (
	ivQuestionJSON = `{"question":"Tell me about your Go work.","competency_tag":"Go"}`
	ivReportJSON   = `{"verdict":"advance","confidence":"high","scores":[{"competency":"Go","score":4.5,"evidence":"built a payments service"}],"recommended_next_step":"Onsite."}`
)

// fakeInterviewStream captures Send calls and supplies a cancellable context.
type fakeInterviewStream struct {
	grpc.ServerStreamingServer[caliberv1.StartInterviewResponse]

	ctx  context.Context //nolint:containedctx // test fake mirrors grpc.ServerStream's stored context
	mu   sync.Mutex
	sent []*caliberv1.StartInterviewResponse
}

func (f *fakeInterviewStream) Context() context.Context { return f.ctx }

func (f *fakeInterviewStream) Send(m *caliberv1.StartInterviewResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakeInterviewStream) messages() []*caliberv1.StartInterviewResponse {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*caliberv1.StartInterviewResponse(nil), f.sent...)
}

func interviewRole(t *testing.T) *role.Role {
	t.Helper()
	r, err := role.NewRole(kernel.NewID(),
		role.RoleSpec{Title: "Backend Engineer", Seniority: role.SeniorityMid},
		role.Rubric{Competencies: []role.Competency{{Name: "Go", Weight: 1, MustHave: true}}},
		time.Unix(1, 0))
	require.NoError(t, err)
	return r
}

// devShapedLLM returns a question or report card depending on the system prompt.
func devShapedLLM(ctrl *gomock.Controller) *mocks.MockLLMClient {
	llm := mocks.NewMockLLMClient(ctrl)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req app.LLMRequest) (app.LLMResponse, error) {
			if strings.Contains(req.System, "score a screening interview") {
				return app.LLMResponse{Text: ivReportJSON}, nil
			}
			return app.LLMResponse{Text: ivQuestionJSON}, nil
		}).AnyTimes()
	return llm
}

func TestStartInterviewStreamsQuestionThenReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl := interviewRole(t)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil).AnyTimes()

	// maxTurns = 1 -> the first answer completes the interview.
	srv := NewInterviewServer(interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 1))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stream := &fakeInterviewStream{ctx: ctx}
	done := make(chan error, 1)
	go func() {
		done <- srv.StartInterview(&caliberv1.StartInterviewRequest{
			RoleId: rl.ID.String(), CandidateId: kernel.NewID().String(), Mode: caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
		}, stream)
	}()

	require.Eventually(t, func() bool { return len(stream.messages()) >= 2 }, 2*time.Second, 5*time.Millisecond,
		"expected status + first question")
	msgs := stream.messages()
	assert.Equal(t, "open", msgs[0].GetStatus().GetState())
	q := msgs[1].GetQuestion()
	require.NotNil(t, q)
	assert.Equal(t, "Go", q.GetCompetencyTag())

	resp, err := srv.SubmitAnswer(context.Background(),
		&caliberv1.SubmitAnswerRequest{InterviewId: q.GetInterviewId(), Answer: "I built a payments service in Go."})
	require.NoError(t, err)
	assert.True(t, resp.GetAccepted())

	select {
	case err := <-done:
		require.NoError(t, err, "stream returns cleanly after the report card")
	case <-time.After(2 * time.Second):
		t.Fatal("StartInterview did not finish after the report card")
	}
	var card *caliberv1.ReportCard
	for _, m := range stream.messages() {
		if m.GetReportCard() != nil {
			card = m.GetReportCard()
		}
	}
	require.NotNil(t, card, "report card streamed")
	assert.Equal(t, caliberv1.InterviewVerdict_INTERVIEW_VERDICT_ADVANCE, card.GetVerdict())
	require.NotEmpty(t, card.GetScores())
	assert.NotEmpty(t, card.GetScores()[0].GetEvidence())
}

func TestGetReportCardHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl := interviewRole(t)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil).AnyTimes()
	interviewer := interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 1)
	srv := NewInterviewServer(interviewer)

	iv, q, err := interviewer.Start(context.Background(), rl.ID, kernel.NewID(), 1) // ModeText
	require.NoError(t, err)
	_, _, err = interviewer.Answer(context.Background(), iv.ID, "concrete Go example")
	require.NoError(t, err)
	_ = q

	resp, err := srv.GetReportCard(context.Background(), &caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, caliberv1.InterviewVerdict_INTERVIEW_VERDICT_ADVANCE, resp.GetReportCard().GetVerdict())
}

func TestGetReportCardNotReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(interviewRole(t), nil).AnyTimes()
	interviewer := interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 4)
	iv, _, err := interviewer.Start(context.Background(), kernel.NewID(), kernel.NewID(), 1)
	require.NoError(t, err)
	_, err = NewInterviewServer(interviewer).GetReportCard(context.Background(),
		&caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInterviewBroker(t *testing.T) {
	b := newInterviewBroker()
	ch := b.subscribe("iv-1")
	b.publish("iv-1", statusEvent("asking", "next"))
	select {
	case msg := <-ch:
		assert.Equal(t, "asking", msg.GetStatus().GetState())
	case <-time.After(time.Second):
		t.Fatal("expected a published message")
	}
	b.publish("unknown", statusEvent("x", "y")) // no subscriber: no panic, no-op
	b.unsubscribe("iv-1")
	b.publish("iv-1", statusEvent("late", "after close")) // must not panic after unsubscribe
}
