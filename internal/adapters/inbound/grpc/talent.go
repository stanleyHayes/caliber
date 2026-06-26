package grpcadapter

import (
	"context"

	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

// TalentServer implements caliberv1.TalentServiceServer (Talent Passport).
type TalentServer struct {
	caliberv1.UnimplementedTalentServiceServer

	builder *profilesapp.ProfileBuilder
}

// NewTalentServer builds the talent gRPC service from its use-case.
func NewTalentServer(builder *profilesapp.ProfileBuilder) *TalentServer { return &TalentServer{builder: builder} }

// CreateProfileFromCV parses a CV + intake into an evidence-linked profile.
func (s *TalentServer) CreateProfileFromCV(
	ctx context.Context, req *caliberv1.CreateProfileFromCVRequest,
) (*caliberv1.CreateProfileFromCVResponse, error) {
	profile, err := s.builder.CreateFromCV(
		ctx, kernel.ID(req.GetCandidateId()), req.GetCvText(), intakeFromProto(req.GetIntake()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.CreateProfileFromCVResponse{Profile: talentProfileToProto(profile)}, nil
}

// GetTalentProfile returns a candidate's talent profile.
func (s *TalentServer) GetTalentProfile(
	ctx context.Context, req *caliberv1.GetTalentProfileRequest,
) (*caliberv1.GetTalentProfileResponse, error) {
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
