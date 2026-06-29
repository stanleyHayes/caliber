// Package grpcadapter maps the generated gRPC services onto application
// use-cases and translates domain types/errors to/from the wire contract.
package grpcadapter

import (
	"log/slog"

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
	case kernel.KindTooManyRequests:
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		// An unclassified (internal) error may carry raw infrastructure detail —
		// pgx/pgconn text, schema or constraint fragments. Log it server-side for
		// diagnosis but return an opaque message so it never reaches the client
		// (CWE-209). The author-controlled kinds above keep their explanatory text.
		slog.Error("grpc internal error", "error", err.Error())
		return status.Error(codes.Internal, "internal error")
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

// exclusionToProto maps a domain hard-filter Exclusion to its proto form.
func exclusionToProto(e matchingdom.Exclusion) *caliberv1.CandidateExclusion {
	return &caliberv1.CandidateExclusion{
		CandidateId: e.CandidateID.String(),
		Gate:        e.Gate,
		Reason:      e.Reason,
	}
}

// specFromProto maps a proto RoleSpec into the domain spec.
func specFromProto(p *caliberv1.RoleSpec) role.RoleSpec {
	var band kernel.SalaryBand
	if sb := p.GetSalaryBand(); sb != nil {
		band = kernel.SalaryBand{Currency: sb.GetCurrency(), Low: sb.GetLow(), High: sb.GetHigh()}
	}
	return role.RoleSpec{
		Title:            p.GetTitle(),
		Location:         p.GetLocation(),
		Seniority:        seniorityFromProto(p.GetSeniority()),
		Availability:     p.GetAvailability(),
		Responsibilities: p.GetResponsibilities(),
		MustHaves:        p.GetMustHaves(),
		NiceToHaves:      p.GetNiceToHaves(),
		SalaryBand:       band,
	}
}

// rubricFromProto maps a proto Rubric into the domain rubric.
func rubricFromProto(p *caliberv1.Rubric) role.Rubric {
	comps := make([]role.Competency, 0, len(p.GetCompetencies()))
	for _, c := range p.GetCompetencies() {
		comps = append(comps, role.Competency{Name: c.GetName(), Weight: c.GetWeight(), MustHave: c.GetMustHave()})
	}
	return role.Rubric{Competencies: comps}
}

func seniorityFromProto(s caliberv1.Seniority) role.Seniority {
	switch s {
	case caliberv1.Seniority_SENIORITY_JUNIOR:
		return role.SeniorityJunior
	case caliberv1.Seniority_SENIORITY_MID:
		return role.SeniorityMid
	case caliberv1.Seniority_SENIORITY_SENIOR:
		return role.SenioritySenior
	case caliberv1.Seniority_SENIORITY_LEAD:
		return role.SeniorityLead
	default:
		return role.SeniorityUnspecified
	}
}

// pageFromProto builds a clamped domain page from a proto PageRequest.
func pageFromProto(p *caliberv1.PageRequest) kernel.Page {
	return kernel.NewPage(max(int(p.GetPage()), 1), int(p.GetPageSize()))
}

// pageResponseToProto builds a proto PageResponse from a page and total count.
func pageResponseToProto(page kernel.Page, total int64) *caliberv1.PageResponse {
	size := page.Limit()
	var totalPages int32
	if size > 0 {
		totalPages = int32((total + int64(size) - 1) / int64(size)) //nolint:gosec // bounded page count
	}
	return &caliberv1.PageResponse{
		Page:       int32(page.Number), //nolint:gosec // bounded page number
		PageSize:   int32(size),        //nolint:gosec // bounded page size (<=100)
		TotalItems: total,
		TotalPages: totalPages,
	}
}
