package grpcadapter

import (
	"context"
	"sync"

	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// interviewBroker fans a candidate's submitted answers (unary SubmitAnswer) out
// to their open StartInterview server-stream: SubmitAnswer publishes the next
// question (or report card) and the stream handler forwards it to the client.
type interviewBroker struct {
	mu       sync.Mutex
	channels map[string]chan *caliberv1.StartInterviewResponse
}

func newInterviewBroker() *interviewBroker {
	return &interviewBroker{channels: map[string]chan *caliberv1.StartInterviewResponse{}}
}

func (b *interviewBroker) subscribe(id string) chan *caliberv1.StartInterviewResponse {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan *caliberv1.StartInterviewResponse, 4)
	b.channels[id] = ch
	return ch
}

func (b *interviewBroker) publish(id string, msg *caliberv1.StartInterviewResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.channels[id]; ok {
		select {
		case ch <- msg:
		default: // buffer full (no live consumer keeping up); drop rather than block
		}
	}
}

func (b *interviewBroker) unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.channels[id]; ok {
		delete(b.channels, id)
		close(ch)
	}
}

// InterviewServer implements caliberv1.InterviewServiceServer (Flow B).
type InterviewServer struct {
	caliberv1.UnimplementedInterviewServiceServer

	interviewer *interviewapp.Interviewer
	broker      *interviewBroker
}

// NewInterviewServer builds the interview gRPC service from its use-case.
func NewInterviewServer(interviewer *interviewapp.Interviewer) *InterviewServer {
	return &InterviewServer{interviewer: interviewer, broker: newInterviewBroker()}
}

// StartInterview opens a session, streams the first question, then forwards each
// subsequent question (and finally the report card) as answers are submitted.
func (s *InterviewServer) StartInterview(
	req *caliberv1.StartInterviewRequest, stream caliberv1.InterviewService_StartInterviewServer,
) error {
	ctx := stream.Context()
	// A candidate takes their own screening (CAL-116).
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return errToStatus(err)
	}
	iv, question, err := s.interviewer.Start(
		ctx, kernel.ID(req.GetRoleId()), kernel.ID(req.GetCandidateId()), interviewModeFromProto(req.GetMode()),
	)
	if err != nil {
		return errToStatus(err)
	}
	ch := s.broker.subscribe(iv.ID.String())
	defer s.broker.unsubscribe(iv.ID.String())

	if err := stream.Send(statusEvent("open", "interview opened")); err != nil {
		return err
	}
	if err := stream.Send(questionEvent(iv.ID, question)); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-ch:
			if err := stream.Send(msg); err != nil {
				return err
			}
			if msg.GetReportCard() != nil {
				return nil
			}
		}
	}
}

// SubmitAnswer records an answer and pushes the next question (or report card)
// onto the candidate's open stream.
func (s *InterviewServer) SubmitAnswer(
	ctx context.Context, req *caliberv1.SubmitAnswerRequest,
) (*caliberv1.SubmitAnswerResponse, error) {
	if _, err := RequireRole(ctx, identity.RoleCandidate); err != nil {
		return nil, errToStatus(err)
	}
	id := kernel.ID(req.GetInterviewId())
	pending, report, err := s.interviewer.Answer(ctx, id, req.GetAnswer())
	if err != nil {
		return nil, errToStatus(err)
	}
	switch {
	case report != nil:
		s.broker.publish(id.String(), reportEvent(report))
	case pending != nil:
		s.broker.publish(id.String(), questionEvent(id, pending))
	}
	return &caliberv1.SubmitAnswerResponse{Accepted: true}, nil
}

// GetReportCard returns a completed interview's report card.
func (s *InterviewServer) GetReportCard(
	ctx context.Context, req *caliberv1.GetReportCardRequest,
) (*caliberv1.GetReportCardResponse, error) {
	if _, err := RequireAuth(ctx); err != nil {
		return nil, errToStatus(err)
	}
	card, err := s.interviewer.Report(ctx, kernel.ID(req.GetInterviewId()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GetReportCardResponse{ReportCard: reportCardToProto(card)}, nil
}

func questionEvent(id kernel.ID, q *interviewdom.PendingQuestion) *caliberv1.StartInterviewResponse {
	return &caliberv1.StartInterviewResponse{Event: &caliberv1.StartInterviewResponse_Question{
		Question: &caliberv1.InterviewQuestion{
			InterviewId:   id.String(),
			Ordinal:       int32(q.Ordinal), //nolint:gosec // ordinal is a small bounded turn index
			Text:          q.Text,
			CompetencyTag: q.CompetencyTag,
		},
	}}
}

func statusEvent(state, message string) *caliberv1.StartInterviewResponse {
	return &caliberv1.StartInterviewResponse{Event: &caliberv1.StartInterviewResponse_Status{
		Status: &caliberv1.InterviewStatusEvent{State: state, Message: message},
	}}
}

func reportEvent(card *interviewdom.ReportCard) *caliberv1.StartInterviewResponse {
	return &caliberv1.StartInterviewResponse{Event: &caliberv1.StartInterviewResponse_ReportCard{
		ReportCard: reportCardToProto(card),
	}}
}

func reportCardToProto(card *interviewdom.ReportCard) *caliberv1.ReportCard {
	scores := make([]*caliberv1.CompetencyScore, 0, len(card.Scores))
	for _, sc := range card.Scores {
		scores = append(scores, &caliberv1.CompetencyScore{Competency: sc.Competency, Score: sc.Score, Evidence: sc.Evidence})
	}
	return &caliberv1.ReportCard{
		InterviewId:         card.InterviewID.String(),
		RoleId:              card.RoleID.String(),
		CandidateId:         card.CandidateID.String(),
		Verdict:             verdictToProto(card.Verdict),
		Confidence:          confidenceToProto(card.Confidence),
		Scores:              scores,
		RecommendedNextStep: card.RecommendedNextStep,
	}
}

func verdictToProto(v interviewdom.InterviewVerdict) caliberv1.InterviewVerdict {
	switch v {
	case interviewdom.VerdictAdvance:
		return caliberv1.InterviewVerdict_INTERVIEW_VERDICT_ADVANCE
	case interviewdom.VerdictHold:
		return caliberv1.InterviewVerdict_INTERVIEW_VERDICT_HOLD
	case interviewdom.VerdictDecline:
		return caliberv1.InterviewVerdict_INTERVIEW_VERDICT_DECLINE
	default:
		return caliberv1.InterviewVerdict_INTERVIEW_VERDICT_UNSPECIFIED
	}
}

func interviewModeFromProto(m caliberv1.InterviewMode) interviewdom.InterviewMode {
	if m == caliberv1.InterviewMode_INTERVIEW_MODE_VOICE {
		return interviewdom.ModeVoice
	}
	return interviewdom.ModeText // TEXT is the reliable default
}
