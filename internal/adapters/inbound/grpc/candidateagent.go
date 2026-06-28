package grpcadapter

import (
	"context"
	"sync"

	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// AgentServer implements caliberv1.CandidateAgentServiceServer (Flow C). It runs
// the agent inline (no async queue yet) and remembers the last wake-up view per
// candidate for GetWakeUpView.
type AgentServer struct {
	caliberv1.UnimplementedCandidateAgentServiceServer

	runner *candidateagentapp.AgentRunner
	apps   agentdom.ApplicationRepository

	mu      sync.Mutex
	wakeups map[string]agentdom.WakeUpView
}

// NewAgentServer builds the candidate-agent gRPC service.
func NewAgentServer(runner *candidateagentapp.AgentRunner, apps agentdom.ApplicationRepository) *AgentServer {
	return &AgentServer{runner: runner, apps: apps, wakeups: map[string]agentdom.WakeUpView{}}
}

// RunAgent runs an agent pass inline and returns a job id (the run is already
// complete; an async queue lands with EPIC-03).
func (s *AgentServer) RunAgent(ctx context.Context, req *caliberv1.RunAgentRequest) (*caliberv1.RunAgentResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	view, err := s.runner.Run(ctx, kernel.ID(req.GetCandidateId()), 0)
	if err != nil {
		return nil, errToStatus(err)
	}
	s.remember(req.GetCandidateId(), view)
	return &caliberv1.RunAgentResponse{JobId: kernel.NewID().String()}, nil
}

// TimeAdvance runs the agent "overnight" and returns the wake-up view.
func (s *AgentServer) TimeAdvance(ctx context.Context, req *caliberv1.TimeAdvanceRequest) (*caliberv1.TimeAdvanceResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	view, err := s.runner.Run(ctx, kernel.ID(req.GetCandidateId()), 0)
	if err != nil {
		return nil, errToStatus(err)
	}
	s.remember(req.GetCandidateId(), view)
	return &caliberv1.TimeAdvanceResponse{WakeUp: wakeUpToProto(view)}, nil
}

// GetWakeUpView returns the last remembered wake-up view (zero if none yet).
func (s *AgentServer) GetWakeUpView(
	ctx context.Context, req *caliberv1.GetWakeUpViewRequest,
) (*caliberv1.GetWakeUpViewResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	s.mu.Lock()
	view := s.wakeups[req.GetCandidateId()]
	s.mu.Unlock()
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

func (s *AgentServer) remember(candidateID string, view agentdom.WakeUpView) {
	s.mu.Lock()
	s.wakeups[candidateID] = view
	s.mu.Unlock()
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
