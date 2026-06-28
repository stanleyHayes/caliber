package grpcadapter

import (
	"context"

	"github.com/xcreativs/caliber/internal/adapters/outbound/cvtext"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// maxCVFileBytes caps an uploaded CV file to bound memory + abuse (CAL-042).
const maxCVFileBytes = 10 << 20 // 10 MiB

// TalentServer implements caliberv1.TalentServiceServer (Talent Passport).
type TalentServer struct {
	caliberv1.UnimplementedTalentServiceServer

	builder *profilesapp.ProfileBuilder
}

// NewTalentServer builds the talent gRPC service from its use-case.
func NewTalentServer(builder *profilesapp.ProfileBuilder) *TalentServer {
	return &TalentServer{builder: builder}
}

// CreateProfileFromCV parses a CV (raw text or an uploaded file) + intake into an
// evidence-linked profile. A candidate builds only their own profile (CAL-116).
func (s *TalentServer) CreateProfileFromCV(
	ctx context.Context, req *caliberv1.CreateProfileFromCVRequest,
) (*caliberv1.CreateProfileFromCVResponse, error) {
	if err := requireSelfCandidate(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	cvText, err := resolveCVText(req)
	if err != nil {
		return nil, errToStatus(err)
	}
	profile, err := s.builder.CreateFromCV(
		ctx, kernel.ID(req.GetCandidateId()), cvText, intakeFromProto(req.GetIntake()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.CreateProfileFromCVResponse{Profile: talentProfileToProto(profile)}, nil
}

// resolveCVText prefers an uploaded file (parsed to plain text, size-capped) over
// raw cv_text; the builder rejects an empty result.
func resolveCVText(req *caliberv1.CreateProfileFromCVRequest) (string, error) {
	file := req.GetCvFile()
	if len(file) == 0 {
		return req.GetCvText(), nil
	}
	if len(file) > maxCVFileBytes {
		return "", kernel.Invalidf("talent: CV file exceeds the %d MiB limit", maxCVFileBytes>>20)
	}
	return cvtext.Extract(req.GetCvFilename(), file)
}

// GetTalentProfile returns a candidate's talent profile, visible to the owning
// candidate or to a reviewer (employers/recruiters view profiles when
// shortlisting) — CAL-116.
func (s *TalentServer) GetTalentProfile(
	ctx context.Context, req *caliberv1.GetTalentProfileRequest,
) (*caliberv1.GetTalentProfileResponse, error) {
	if err := requireSelfCandidateOrReviewer(ctx, req.GetCandidateId()); err != nil {
		return nil, errToStatus(err)
	}
	profile, err := s.builder.GetProfile(ctx, kernel.ID(req.GetCandidateId()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GetTalentProfileResponse{Profile: talentProfileToProto(profile)}, nil
}

func intakeFromProto(p *caliberv1.CandidateIntake) talent.CandidateIntake {
	if p == nil {
		return talent.CandidateIntake{}
	}
	return talent.CandidateIntake{
		TargetTitles: p.GetTargetTitles(),
		Location:     p.GetLocation(),
		SalaryFloor:  p.GetSalaryFloor(),
		DealBreakers: p.GetDealBreakers(),
	}
}

func talentProfileToProto(p *talent.TalentProfile) *caliberv1.TalentProfile {
	comps := make([]*caliberv1.ProfileCompetency, 0, len(p.Competencies))
	for _, c := range p.Competencies {
		comps = append(comps, &caliberv1.ProfileCompetency{
			Name:          c.Name,
			Level:         c.Level,
			EvidenceQuote: c.EvidenceQuote,
			SourceSpan:    c.SourceSpan,
		})
	}
	return &caliberv1.TalentProfile{
		Id:             p.ID.String(),
		CandidateId:    p.CandidateID.String(),
		Summary:        p.Summary,
		Competencies:   comps,
		PassportStatus: passportStatusToProto(p.PassportStatus),
	}
}
