package grpcadapter

import (
	"context"

	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

const defaultShortlistLimit = 20

// MatchServer implements caliberv1.MatchingServiceServer (Flow A shortlist).
type MatchServer struct {
	caliberv1.UnimplementedMatchingServiceServer

	shortlister *matchingapp.Shortlister
	refiner     *matchingapp.Refiner
}

// NewMatchServer builds the matching gRPC service from its use-cases.
func NewMatchServer(shortlister *matchingapp.Shortlister, refiner *matchingapp.Refiner) *MatchServer {
	return &MatchServer{shortlister: shortlister, refiner: refiner}
}

// GenerateShortlist returns an explainable ranked shortlist for a role.
func (s *MatchServer) GenerateShortlist(
	ctx context.Context, req *caliberv1.GenerateShortlistRequest,
) (*caliberv1.GenerateShortlistResponse, error) {
	result, err := s.shortlister.GenerateShortlist(ctx, kernel.ID(req.GetRoleId()), pageLimit(req.GetPage()))
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.GenerateShortlistResponse{Shortlist: shortlistToProto(result)}, nil
}

// RefineShortlist applies edited spec/rubric overrides to the role and re-ranks.
func (s *MatchServer) RefineShortlist(
	ctx context.Context, req *caliberv1.RefineShortlistRequest,
) (*caliberv1.RefineShortlistResponse, error) {
	result, err := s.refiner.Refine(
		ctx, kernel.ID(req.GetRoleId()), specFromProto(req.GetSpec()), rubricFromProto(req.GetRubric()), pageLimit(req.GetPage()),
	)
	if err != nil {
		return nil, errToStatus(err)
	}
	return &caliberv1.RefineShortlistResponse{Shortlist: shortlistToProto(result)}, nil
}

func pageLimit(p *caliberv1.PageRequest) int {
	if limit := int(p.GetPageSize()); limit > 0 {
		return limit
	}
	return defaultShortlistLimit
}

func shortlistToProto(result *matchingapp.ShortlistResult) *caliberv1.Shortlist {
	protoMatches := make([]*caliberv1.Match, 0, len(result.Matches))
	for _, m := range result.Matches {
		protoMatches = append(protoMatches, matchToProto(m))
	}
	exclusions := make([]*caliberv1.CandidateExclusion, 0, len(result.Exclusions))
	for _, e := range result.Exclusions {
		exclusions = append(exclusions, exclusionToProto(e))
	}
	return &caliberv1.Shortlist{
		Matches:    protoMatches,
		PoolDepth:  int32(len(protoMatches)), //nolint:gosec // shortlist length is small and bounded
		Exclusions: exclusions,
	}
}
