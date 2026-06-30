package grpcadapter

import (
	"context"
	"strconv"
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
	"github.com/xcreativs/caliber/internal/domain/identity"
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

	candidateID := kernel.NewID()
	ctx, cancel := context.WithCancel(asCandidate(t.Context(), candidateID))
	defer cancel()
	stream := &fakeInterviewStream{ctx: ctx}
	done := make(chan error, 1)
	go func() {
		done <- srv.StartInterview(&caliberv1.StartInterviewRequest{
			RoleId: rl.ID.String(), CandidateId: candidateID.String(), Mode: caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
		}, stream)
	}()

	require.Eventually(t, func() bool { return len(stream.messages()) >= 2 }, 2*time.Second, 5*time.Millisecond,
		"expected status + first question")
	msgs := stream.messages()
	assert.Equal(t, "open", msgs[0].GetStatus().GetState())
	q := msgs[1].GetQuestion()
	require.NotNil(t, q)
	assert.Equal(t, "Go", q.GetCompetencyTag())

	resp, err := srv.SubmitAnswer(asCandidate(context.Background(), candidateID),
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

func TestStartInterviewStreamCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl := interviewRole(t)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil).AnyTimes()
	srv := NewInterviewServer(interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 4))

	candidateID := kernel.NewID()
	ctx, cancel := context.WithCancel(asCandidate(t.Context(), candidateID))
	stream := &fakeInterviewStream{ctx: ctx}
	done := make(chan error, 1)
	go func() {
		done <- srv.StartInterview(&caliberv1.StartInterviewRequest{
			RoleId: rl.ID.String(), CandidateId: candidateID.String(), Mode: caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
		}, stream)
	}()

	require.Eventually(t, func() bool { return len(stream.messages()) >= 2 }, 2*time.Second, 5*time.Millisecond,
		"expected status + first question before cancellation")
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("StartInterview did not return after context cancellation")
	}
}

// slowInterviewStream passes the first blockAfter messages through, then blocks
// on unblockCh until it is closed, simulating a slow consumer.
type slowInterviewStream struct {
	grpc.ServerStreamingServer[caliberv1.StartInterviewResponse]
	ctx        context.Context
	mu         sync.Mutex
	sent       []*caliberv1.StartInterviewResponse
	blockAfter int
	unblockCh  chan struct{}
}

func (s *slowInterviewStream) Context() context.Context { return s.ctx }

func (s *slowInterviewStream) Send(m *caliberv1.StartInterviewResponse) error {
	s.mu.Lock()
	count := len(s.sent)
	s.mu.Unlock()
	if count >= s.blockAfter {
		<-s.unblockCh
	}
	s.mu.Lock()
	s.sent = append(s.sent, m)
	s.mu.Unlock()
	return nil
}

func (s *slowInterviewStream) messages() []*caliberv1.StartInterviewResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*caliberv1.StartInterviewResponse(nil), s.sent...)
}

func TestStartInterviewStreamBackpressureSlowConsumer(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl := interviewRole(t)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil).AnyTimes()
	srv := NewInterviewServer(interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 4))

	candidateID := kernel.NewID()
	ctx, cancel := context.WithCancel(asCandidate(t.Context(), candidateID))
	defer cancel()
	stream := &slowInterviewStream{ctx: ctx, blockAfter: 2, unblockCh: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		done <- srv.StartInterview(&caliberv1.StartInterviewRequest{
			RoleId: rl.ID.String(), CandidateId: candidateID.String(), Mode: caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
		}, stream)
	}()

	require.Eventually(t, func() bool { return len(stream.messages()) >= 2 }, 2*time.Second, 5*time.Millisecond,
		"expected status + first question before blocking")

	var interviewID string
	for _, m := range stream.messages() {
		if q := m.GetQuestion(); q != nil {
			interviewID = q.GetInterviewId()
			break
		}
	}
	require.NotEmpty(t, interviewID)

	const published = 20
	accepted := 0
	for i := 0; i < published; i++ {
		if srv.broker.publish(interviewID, statusEvent("status", strconv.Itoa(i))) {
			accepted++
		}
	}
	assert.Less(t, accepted, published, "some events should be dropped under backpressure")
	assert.GreaterOrEqual(t, accepted, srv.broker.capacity, "buffer should accept up to capacity events")

	close(stream.unblockCh)

	require.Eventually(t, func() bool { return len(stream.messages()) >= 2+accepted }, 2*time.Second, 5*time.Millisecond,
		"stream should drain buffered events after the consumer unblocks")

	// The stream handler loops waiting for more events; cancel to make it return.
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not return after cancellation")
	}

	total := len(stream.messages())
	assert.GreaterOrEqual(t, total, 2)
	assert.Less(t, total, 2+published, "slow consumer should not receive every published event")
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

	// The employer who owns the role may read the report card.
	resp, err := srv.GetReportCard(asEmployer(context.Background(), rl.EmployerID),
		&caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, caliberv1.InterviewVerdict_INTERVIEW_VERDICT_ADVANCE, resp.GetReportCard().GetVerdict())
}

// TestGetReportCardOwnershipIDOR locks CAL-116: a report card (the Flow B
// verdict/scores/evidence) is private to its owning candidate and the employer
// who owns the role — not every reviewer, and not every logged-in user.
func TestGetReportCardOwnershipIDOR(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl := interviewRole(t)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil).AnyTimes()
	interviewer := interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 1)
	srv := NewInterviewServer(interviewer)

	candidateID := kernel.NewID()
	iv, _, err := interviewer.Start(context.Background(), rl.ID, candidateID, 1)
	require.NoError(t, err)
	_, _, err = interviewer.Answer(context.Background(), iv.ID, "concrete Go example")
	require.NoError(t, err)
	req := &caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()}

	// A different employer (does not own the role) is denied.
	_, err = srv.GetReportCard(asEmployer(context.Background(), kernel.NewID()), req)
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "a non-owning employer cannot read the card")

	// A different candidate is denied; the owning candidate is allowed.
	_, err = srv.GetReportCard(asCandidate(context.Background(), kernel.NewID()), req)
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "another candidate cannot read the card")

	resp, err := srv.GetReportCard(asCandidate(context.Background(), candidateID), req)
	require.NoError(t, err, "the owning candidate may read their own card")
	assert.Equal(t, caliberv1.InterviewVerdict_INTERVIEW_VERDICT_ADVANCE, resp.GetReportCard().GetVerdict())
}

func TestGetReportCardNotReady(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(interviewRole(t), nil).AnyTimes()
	interviewer := interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 4)
	iv, _, err := interviewer.Start(context.Background(), kernel.NewID(), kernel.NewID(), 1)
	require.NoError(t, err)
	_, err = NewInterviewServer(interviewer).GetReportCard(asRole(context.Background(), identity.RoleEmployer),
		&caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestInterviewBroker(t *testing.T) {
	b := newInterviewBroker()
	ch := b.subscribe("iv-1")
	require.True(t, b.publish("iv-1", statusEvent("asking", "next")))
	select {
	case msg := <-ch:
		assert.Equal(t, "asking", msg.GetStatus().GetState())
	case <-time.After(time.Second):
		t.Fatal("expected a published message")
	}
	assert.False(t, b.publish("unknown", statusEvent("x", "y")), "no subscriber: dropped")
	b.unsubscribe("iv-1")
	assert.False(t, b.publish("iv-1", statusEvent("late", "after close")), "must not panic after unsubscribe")
}

func TestInterviewBrokerBackpressureDropsWhenFull(t *testing.T) {
	b := newInterviewBrokerWithCapacity(2)
	ch := b.subscribe("iv-1")
	assert.True(t, b.publish("iv-1", statusEvent("a", "1")), "first publish accepted")
	assert.True(t, b.publish("iv-1", statusEvent("a", "2")), "second publish accepted")
	assert.False(t, b.publish("iv-1", statusEvent("a", "3")), "third publish drops under backpressure")
	assert.Equal(t, "a", (<-ch).GetStatus().GetState())
	assert.Equal(t, "a", (<-ch).GetStatus().GetState())
}

func TestInterviewBrokerConcurrentPublishUnsubscribe(t *testing.T) {
	b := newInterviewBrokerWithCapacity(4)
	_ = b.subscribe("iv-1")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b.publish("iv-1", statusEvent("status", strconv.Itoa(i)))
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		b.unsubscribe("iv-1")
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// OK: no panic despite concurrent publish/unsubscribe.
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent publish/unsubscribe did not complete")
	}
}

func TestInterviewAuthz(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(interviewRole(t), nil).AnyTimes()
	srv := NewInterviewServer(interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 1))

	// StartInterview: a candidate may only screen as themselves.
	other := &fakeInterviewStream{ctx: asCandidate(t.Context(), kernel.NewID())}
	err := srv.StartInterview(&caliberv1.StartInterviewRequest{
		RoleId: kernel.NewID().String(), CandidateId: kernel.NewID().String(), Mode: caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
	}, other)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	anon := &fakeInterviewStream{ctx: t.Context()}
	err = srv.StartInterview(&caliberv1.StartInterviewRequest{
		RoleId: kernel.NewID().String(), CandidateId: kernel.NewID().String(), Mode: caliberv1.InterviewMode_INTERVIEW_MODE_TEXT,
	}, anon)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// SubmitAnswer is candidate-only; GetReportCard requires authentication.
	_, err = srv.SubmitAnswer(asRole(context.Background(), identity.RoleEmployer),
		&caliberv1.SubmitAnswerRequest{InterviewId: kernel.NewID().String(), Answer: "x"})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = srv.GetReportCard(context.Background(), &caliberv1.GetReportCardRequest{InterviewId: kernel.NewID().String()})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestInterviewOwnership(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := mocks.NewMockRoleRepository(ctrl)
	rl := interviewRole(t)
	roles.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(rl, nil).AnyTimes()
	interviewer := interviewapp.NewInterviewer(roles, memory.NewInterviewRepo(), devShapedLLM(ctrl), 1)
	srv := NewInterviewServer(interviewer)

	owner := kernel.NewID()
	iv, _, err := interviewer.Start(context.Background(), rl.ID, owner, 1) // ModeText
	require.NoError(t, err)

	// A different candidate cannot submit answers to someone else's interview.
	_, err = srv.SubmitAnswer(asCandidate(context.Background(), kernel.NewID()),
		&caliberv1.SubmitAnswerRequest{InterviewId: iv.ID.String(), Answer: "x"})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// The owning candidate can.
	_, err = srv.SubmitAnswer(asCandidate(context.Background(), owner),
		&caliberv1.SubmitAnswerRequest{InterviewId: iv.ID.String(), Answer: "I shipped a Go service."})
	require.NoError(t, err)

	// The report card is now ready. A different candidate cannot read it, and a
	// reviewer who does NOT own the role cannot either (CAL-116) — only the role's
	// owning employer (and the owning candidate) may.
	_, err = srv.GetReportCard(asCandidate(context.Background(), kernel.NewID()),
		&caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = srv.GetReportCard(asEmployer(context.Background(), kernel.NewID()),
		&caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "a non-owning employer is denied")
	_, err = srv.GetReportCard(asEmployer(context.Background(), rl.EmployerID),
		&caliberv1.GetReportCardRequest{InterviewId: iv.ID.String()})
	require.NoError(t, err, "the role's owning employer may read the card")
}
