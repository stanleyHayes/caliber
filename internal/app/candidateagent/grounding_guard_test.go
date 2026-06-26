package candidateagent_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// TestRunRejectsFabricatedSummary is the CAL-071 acceptance guard: the agent's
// tailored content must trace to the verified profile. The model returns a
// summary that claims Kubernetes — a role competency the candidate's profile does
// NOT evidence — so the application is rejected (never submitted) even though the
// candidate is eligible (the profile covers the Go must-have), and the rejection
// is surfaced to the candidate as an explainable highlight rather than dropped
// silently.
func TestRunRejectsFabricatedSummary(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := newDeps(ctrl)
	cid := kernel.NewID()
	profile := profileWith(t, cid, talent.ProfileCompetency{Name: "Go", Level: 4, EvidenceQuote: "built services"})
	rl := openRole(t, []role.Competency{
		{Name: "Go", Weight: 0.5, MustHave: true},
		{Name: "Kubernetes", Weight: 0.5}, // nice-to-have the candidate lacks
	})

	d.candidates.EXPECT().ByID(gomock.Any(), cid).Return(candidate(t, "Accra"), nil)
	d.profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(profile, nil)
	d.roles.EXPECT().ListOpen(gomock.Any(), gomock.Any()).Return([]*role.Role{rl}, int64(1), nil)
	d.llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(
		app.LLMResponse{Text: `{"fit_score":0.9,"apply":true,"tailored_summary":"Expert in Go and Kubernetes orchestration."}`}, nil)
	// apps.Create must NOT be called — the fabricated application is never submitted.

	view, err := d.runner().Run(context.Background(), cid, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, view.NewMatches, "eligible: the profile covers the Go must-have")
	assert.Zero(t, view.ApplicationsSubmitted, "fabricated Kubernetes claim is rejected, not applied")
	require.Len(t, view.Highlights, 1, "the rejection is surfaced, not silent")
	assert.Contains(t, view.Highlights[0], "Skipped")
	assert.Contains(t, view.Highlights[0], "Kubernetes", "names the unverified skill that triggered the rejection")
}
