package grpcadapter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

func TestTalentCreateThenGetProfile(t *testing.T) {
	ctx := context.Background()
	candidates := memory.NewCandidateRepo()
	profiles := memory.NewTalentProfileRepo()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, candidates.Create(ctx, cand))

	srv := NewTalentServer(profilesapp.NewProfileBuilder(candidates, profiles, llm.NewDev()))

	resp, err := srv.CreateProfileFromCV(ctx, &caliberv1.CreateProfileFromCVRequest{
		CandidateId: cand.ID.String(),
		CvText:      "Senior engineer experienced in Go and Postgres at scale, with gRPC services.",
		Intake:      &caliberv1.CandidateIntake{Location: "Accra"},
	})
	require.NoError(t, err)
	names := map[string]bool{}
	for _, c := range resp.GetProfile().GetCompetencies() {
		names[c.GetName()] = true
		assert.NotEmpty(t, c.GetEvidenceQuote(), "every competency cites evidence")
	}
	assert.True(t, names["Go"] && names["Postgres"], "extracted from the CV's actual content")

	got, err := srv.GetTalentProfile(ctx, &caliberv1.GetTalentProfileRequest{CandidateId: cand.ID.String()})
	require.NoError(t, err)
	assert.Len(t, got.GetProfile().GetCompetencies(), len(resp.GetProfile().GetCompetencies()))
}
