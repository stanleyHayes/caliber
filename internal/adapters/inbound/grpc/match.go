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
}

// NewMatchServer builds the matching gRPC service from its use-case.
func NewMatchServer(s *matchingapp.Shortlister) *MatchServer { return &MatchServer{shortlister: s} }

// GenerateShortlist returns an explainable ranked shortlist for a role.
func (s *MatchServer) GenerateShortlist(
	ctx context.Context, req *caliberv1.GenerateShortlistRequest,
) (*caliberv1.GenerateShortlistResponse, error) {
	limit := int(req.GetPage().GetPageSize())
	if limit <= 0 {
		limit = defaultShortlistLimit
	}
	matches, err := s.shortlister.GenerateShortlist(ctx, kernel.ID(req.GetRoleId()), limit)
	if err != nil {
		return nil, errToStatus(err)
	}
	protoMatches := make([]*caliberv1.Match, 0, len(matches))
	for _, m := range matches {
		protoMatches = append(protoMatches, matchToProto(m))
	}
	return &caliberv1.GenerateShortlistResponse{
		Shortlist: &caliberv1.Shortlist{
			Matches:   protoMatches,
			PoolDepth: int32(len(protoMatches)), //nolint:gosec // shortlist length is small and bounded
		},
	}, nil
}
