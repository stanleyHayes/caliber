// Package grpcadapter maps the generated gRPC services onto application
// use-cases and translates domain types/errors to/from the wire contract.
package grpcadapter

import (
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/role"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// errToStatus maps a domain error kind to a gRPC status.
func errToStatus(err error) error {
	switch kernel.KindOf(err) {
	case kernel.KindInvalid:
		return status.Error(codes.InvalidArgument, err.Error())
	case kernel.KindNotFound:
		return status.Error(codes.NotFound, err.Error())
	case kernel.KindConflict:
		return status.Error(codes.AlreadyExists, err.Error())
	case kernel.KindUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case kernel.KindForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func seniorityToProto(s role.Seniority) caliberv1.Seniority {
	switch s {
	case role.SeniorityJunior:
		return caliberv1.Seniority_SENIORITY_JUNIOR
	case role.SeniorityMid:
		return caliberv1.Seniority_SENIORITY_MID
	case role.SenioritySenior:
		return caliberv1.Seniority_SENIORITY_SENIOR
	case role.SeniorityLead:
		return caliberv1.Seniority_SENIORITY_LEAD
	default:
		return caliberv1.Seniority_SENIORITY_UNSPECIFIED
	}
}

func roleStatusToProto(s role.RoleStatus) caliberv1.RoleStatus {
	switch s {
	case role.RoleDraft:
		return caliberv1.RoleStatus_ROLE_STATUS_DRAFT
	case role.RoleOpen:
		return caliberv1.RoleStatus_ROLE_STATUS_OPEN
	case role.RoleClosed:
		return caliberv1.RoleStatus_ROLE_STATUS_CLOSED
	default:
		return caliberv1.RoleStatus_ROLE_STATUS_UNSPECIFIED
	}
}

func specToProto(s role.RoleSpec) *caliberv1.RoleSpec {
	return &caliberv1.RoleSpec{
		Title:            s.Title,
		Location:         s.Location,
		Seniority:        seniorityToProto(s.Seniority),
		Availability:     s.Availability,
		Responsibilities: s.Responsibilities,
		MustHaves:        s.MustHaves,
		NiceToHaves:      s.NiceToHaves,
		SalaryBand: &caliberv1.SalaryBand{
			Currency: s.SalaryBand.Currency,
			Low:      s.SalaryBand.Low,
			High:     s.SalaryBand.High,
		},
	}
}

func rubricToProto(r role.Rubric) *caliberv1.Rubric {
	comps := make([]*caliberv1.Competency, 0, len(r.Competencies))
	for _, c := range r.Competencies {
		comps = append(comps, &caliberv1.Competency{Name: c.Name, Weight: c.Weight, MustHave: c.MustHave})
	}
	return &caliberv1.Rubric{Competencies: comps}
}

func roleToProto(r *role.Role) *caliberv1.Role {
	return &caliberv1.Role{
		Id:         r.ID.String(),
		EmployerId: r.EmployerID.String(),
		Title:      r.Title,
		Status:     roleStatusToProto(r.Status),
		Spec:       specToProto(r.Spec),
		Rubric:     rubricToProto(r.Rubric),
		CreatedAt:  timestamppb.New(r.CreatedAt),
	}
}

func confidenceToProto(c kernel.Confidence) caliberv1.Confidence {
	switch c {
	case kernel.ConfidenceLow:
		return caliberv1.Confidence_CONFIDENCE_LOW
	case kernel.ConfidenceMedium:
		return caliberv1.Confidence_CONFIDENCE_MEDIUM
	case kernel.ConfidenceHigh:
		return caliberv1.Confidence_CONFIDENCE_HIGH
	default:
		return caliberv1.Confidence_CONFIDENCE_UNSPECIFIED
	}
}

func matchToProto(m *matchingdom.Match) *caliberv1.Match {
	breakdown := make([]*caliberv1.MatchBreakdownItem, 0, len(m.Breakdown))
	for _, b := range m.Breakdown {
		breakdown = append(breakdown, &caliberv1.MatchBreakdownItem{
			Competency: b.Competency,
			Score:      b.Score,
			Evidence:   b.Evidence,
		})
	}
	return &caliberv1.Match{
		Id:           m.ID.String(),
		RoleId:       m.RoleID.String(),
		CandidateId:  m.CandidateID.String(),
		OverallScore: m.OverallScore,
		Confidence:   confidenceToProto(m.Confidence),
		Breakdown:    breakdown,
		Rationale:    m.Rationale,
		WatchOuts:    m.WatchOuts,
		ThinEvidence: m.ThinEvidence,
	}
}
