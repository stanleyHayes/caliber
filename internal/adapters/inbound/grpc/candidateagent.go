package grpcadapter

import (
	"context"

	queueadapter "github.com/xcreativs/caliber/internal/adapters/outbound/queue"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	appqueue "github.com/xcreativs/caliber/internal/app/queue"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// AgentServer implements caliberv1.CandidateAgentServiceServer (Flow C).
// When a real task dispatcher is wired, RunAgent/TimeAdvance enqueue background
// work; otherwise they fall back to the synchronous dev path.
type AgentServer struct {
	caliberv1.UnimplementedCandidateAgentServiceServer

	runner     *candidateagentapp.AgentRunner
	apps       agentdom.ApplicationRepository
	dispatcher appqueue.TaskDispatcher
}

// NewAgentServer builds the candidate-agent gRPC service.
func NewAgentServer(
	runner *candidateagentapp.AgentRunner,
	apps agentdom.ApplicationRepository,
	dispatcher appqueue.TaskDispatcher,
) *AgentServer {
	if dispatcher == nil {
		dispatcher = queueadapter.NewNoop()
	}
	return &AgentServer{runner: runner, apps: apps, dispatcher: dispatcher}
}

// RunAgent enqueues a candidate-agent run when a dispatcher is available and
// returns the real task ID; in the dev path it runs synchronously.
func (s *AgentServer) RunAgent(ctx context.Context, req *caliberv1.RunAgentRequest) (*caliberv1.RunAgentResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	candidateID := kernel.ID(req.GetCandidateId())

	if queueadapter.IsNoop(s.dispatcher) {
		if _, err := s.runner.Run(ctx, candidateID, 0); err != nil {
			return nil, errToStatus(err)
		}
		return &caliberv1.RunAgentResponse{JobId: ""}, nil
	}

	taskID, err := s.dispatcher.DispatchCandidateAgentRun(ctx, candidateID)
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.RunAgentResponse{JobId: taskID}, nil
}

// TimeAdvance enqueues a candidate-agent run (the "overnight" demo action) and
// returns the current wake-up view. When no dispatcher is wired it runs inline.
func (s *AgentServer) TimeAdvance(ctx context.Context, req *caliberv1.TimeAdvanceRequest) (*caliberv1.TimeAdvanceResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	candidateID := kernel.ID(req.GetCandidateId())

	if queueadapter.IsNoop(s.dispatcher) {
		view, err := s.runner.Run(ctx, candidateID, 0)
		if err != nil {
			return nil, errToStatus(err)
		}
		return &caliberv1.TimeAdvanceResponse{WakeUp: wakeUpToProto(view)}, nil
	}

	if _, err := s.dispatcher.DispatchCandidateAgentRun(ctx, candidateID); err != nil {
		return nil, errToStatus(err)
	}
	view, _ := s.runner.WakeUpView(ctx, candidateID)
	return &caliberv1.TimeAdvanceResponse{WakeUp: wakeUpToProto(view)}, nil
}

// GetWakeUpView returns the live wake-up view derived from current data.
func (s *AgentServer) GetWakeUpView(
	ctx context.Context, req *caliberv1.GetWakeUpViewRequest,
) (*caliberv1.GetWakeUpViewResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	view, err := s.runner.WakeUpView(ctx, kernel.ID(req.GetCandidateId()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GetWakeUpViewResponse{WakeUp: wakeUpToProto(view)}, nil
}

// ListApplications returns a candidate's applications, newest first, paginated.
func (s *AgentServer) ListApplications(
	ctx context.Context, req *caliberv1.ListApplicationsRequest,
) (*caliberv1.ListApplicationsResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	page := pageFromProto(req.GetPage())
	apps, total, err := s.apps.ByCandidate(ctx, kernel.ID(req.GetCandidateId()), page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.Application, 0, len(apps))
	for _, a := range apps {
		out = append(out, applicationToProto(a))
	}
	return &caliberv1.ListApplicationsResponse{Applications: out, Page: pageResponseToProto(page, total)}, nil
}

func wakeUpToProto(v agentdom.WakeUpView) *caliberv1.WakeUpView {
	return &caliberv1.WakeUpView{
		NewMatches:            int32(v.NewMatches),            //nolint:gosec // small bounded counts
		ApplicationsSubmitted: int32(v.ApplicationsSubmitted), //nolint:gosec // small bounded counts
		ScreeningsCompleted:   int32(v.ScreeningsCompleted),   //nolint:gosec // small bounded counts
		EmployersInterested:   int32(v.EmployersInterested),   //nolint:gosec // small bounded counts
		Highlights:            v.Highlights,
	}
}

func applicationToProto(a *agentdom.Application) *caliberv1.Application {
	return &caliberv1.Application{
		Id:              a.ID.String(),
		RoleId:          a.RoleID.String(),
		CandidateId:     a.CandidateID.String(),
		Source:          appSourceToProto(a.Source),
		TailoredSummary: a.TailoredSummary,
		Status:          appStatusToProto(a.Status),
	}
}

func appSourceToProto(s agentdom.ApplicationSource) caliberv1.ApplicationSource {
	switch s {
	case agentdom.SourceManual:
		return caliberv1.ApplicationSource_APPLICATION_SOURCE_MANUAL
	case agentdom.SourceAgent:
		return caliberv1.ApplicationSource_APPLICATION_SOURCE_AGENT
	default:
		return caliberv1.ApplicationSource_APPLICATION_SOURCE_UNSPECIFIED
	}
}

func appStatusToProto(s agentdom.ApplicationStatus) caliberv1.ApplicationStatus {
	switch s {
	case agentdom.StatusDrafted:
		return caliberv1.ApplicationStatus_APPLICATION_STATUS_DRAFTED
	case agentdom.StatusSubmitted:
		return caliberv1.ApplicationStatus_APPLICATION_STATUS_SUBMITTED
	case agentdom.StatusScreening:
		return caliberv1.ApplicationStatus_APPLICATION_STATUS_SCREENING
	case agentdom.StatusScreened:
		return caliberv1.ApplicationStatus_APPLICATION_STATUS_SCREENED
	default:
		return caliberv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED
	}
}
