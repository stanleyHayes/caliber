package grpcadapter

import (
	"context"

	contestapp "github.com/xcreativs/caliber/internal/app/contest"
	contestdom "github.com/xcreativs/caliber/internal/domain/contest"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ContestServer implements caliberv1.ContestServiceServer (CAL-083): a candidate
// disputes an assessment and a reviewer resolves it. The acting principal is read
// from the authenticated context, never trusted from the request body — a
// candidate can only contest as themselves.
type ContestServer struct {
	caliberv1.UnimplementedContestServiceServer

	svc *contestapp.Service
}

// NewContestServer builds the contest gRPC service.
func NewContestServer(svc *contestapp.Service) *ContestServer { return &ContestServer{svc: svc} }

// RaiseContest opens a contest on behalf of the authenticated candidate.
func (s *ContestServer) RaiseContest(
	ctx context.Context, req *caliberv1.RaiseContestRequest,
) (*caliberv1.RaiseContestResponse, error) {
	principal, err := RequireRole(ctx, identity.RoleCandidate)
	if err != nil {
		return nil, errToStatus(err)
	}
	subject, err := subjectFromProto(req.GetSubject())
	if err != nil {
		return nil, errToStatus(err)
	}
	c, err := s.svc.Raise(ctx, principal.UserID, kernel.ID(req.GetSubjectId()), subject, req.GetReason())
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.RaiseContestResponse{Contest: contestToProto(c)}, nil
}

// ListMyContests returns the authenticated candidate's contests, newest first.
func (s *ContestServer) ListMyContests(
	ctx context.Context, req *caliberv1.ListMyContestsRequest,
) (*caliberv1.ListMyContestsResponse, error) {
	principal, err := RequireAuth(ctx)
	if err != nil {
		return nil, errToStatus(err)
	}
	page := pageFromProto(req.GetPage())
	list, total, err := s.svc.ListForCandidate(ctx, principal.UserID, page)
	if err != nil {
		return nil, errToStatus(err)
	}
	out := make([]*caliberv1.Contest, 0, len(list))
	for _, c := range list {
		out = append(out, contestToProto(c))
	}
	return &caliberv1.ListMyContestsResponse{Contests: out, Page: pageResponseToProto(page, total)}, nil
}

// ResolveContest resolves an open contest as a reviewer (employer/recruiter).
func (s *ContestServer) ResolveContest(
	ctx context.Context, req *caliberv1.ResolveContestRequest,
) (*caliberv1.ResolveContestResponse, error) {
	principal, err := RequireRole(ctx, identity.RoleEmployer, identity.RoleRecruiter)
	if err != nil {
		return nil, errToStatus(err)
	}
	c, err := s.svc.Resolve(ctx, principal.UserID, kernel.ID(req.GetContestId()), req.GetUphold(), req.GetNote())
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.ResolveContestResponse{Contest: contestToProto(c)}, nil
}

func subjectFromProto(s caliberv1.ContestSubject) (contestdom.Subject, error) {
	switch s {
	case caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH:
		return contestdom.SubjectMatch, nil
	case caliberv1.ContestSubject_CONTEST_SUBJECT_REPORT_CARD:
		return contestdom.SubjectReportCard, nil
	default:
		return contestdom.SubjectUnspecified, kernel.Invalid("contest: a valid subject is required")
	}
}

func subjectToProto(s contestdom.Subject) caliberv1.ContestSubject {
	switch s {
	case contestdom.SubjectMatch:
		return caliberv1.ContestSubject_CONTEST_SUBJECT_MATCH
	case contestdom.SubjectReportCard:
		return caliberv1.ContestSubject_CONTEST_SUBJECT_REPORT_CARD
	default:
		return caliberv1.ContestSubject_CONTEST_SUBJECT_UNSPECIFIED
	}
}

func statusToProto(s contestdom.Status) caliberv1.ContestStatus {
	switch s {
	case contestdom.StatusUpheld:
		return caliberv1.ContestStatus_CONTEST_STATUS_UPHELD
	case contestdom.StatusDismissed:
		return caliberv1.ContestStatus_CONTEST_STATUS_DISMISSED
	default:
		return caliberv1.ContestStatus_CONTEST_STATUS_OPEN
	}
}

func contestToProto(c *contestdom.Contest) *caliberv1.Contest {
	out := &caliberv1.Contest{
		Id:          c.ID.String(),
		CandidateId: c.CandidateID.String(),
		Subject:     subjectToProto(c.Subject),
		SubjectId:   c.SubjectID.String(),
		Reason:      c.Reason,
		Status:      statusToProto(c.Status),
		Resolution:  c.Resolution,
		CreatedAt:   timestamppb.New(c.CreatedAt),
	}
	if !c.ResolvedAt.IsZero() {
		out.ResolvedAt = timestamppb.New(c.ResolvedAt)
	}
	return out
}
