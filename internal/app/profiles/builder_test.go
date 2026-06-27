package profiles_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

const extractJSON = `{"summary":"Senior engineer.","competencies":[{"name":"Go","level":4,"evidence_quote":"built services in Go","source_span":"line 3"}]}`

func TestCreateFromCVCreatesProfileAndMergesIntake(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: extractJSON}, nil)
	var updatedCand *talent.Candidate
	candidates.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *talent.Candidate) error { updatedCand = c; return nil })
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	var created *talent.TalentProfile
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *talent.TalentProfile) error { created = p; return nil })

	out, err := profilesapp.NewProfileBuilder(candidates, profiles, llm).
		CreateFromCV(context.Background(), cid, "I built services in Go", talent.CandidateIntake{Location: "Accra", SalaryFloor: 9000})
	require.NoError(t, err)
	require.Len(t, out.Competencies, 1)
	assert.Equal(t, "Go", out.Competencies[0].Name)
	assert.Equal(t, cid, out.CandidateID)
	require.NotNil(t, created)
	require.NotNil(t, updatedCand)
	assert.Equal(t, "Accra", updatedCand.Location, "intake location merged into the candidate")
	assert.InDelta(t, 9000.0, updatedCand.Intake.SalaryFloor, 0.01)
}

func TestCreateFromCVDropsUnevidencedCompetencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)

	// The model returns two competencies; the second has no evidence quote — a
	// no-fabrication violation that must never enter the verified profile.
	const mixed = `{"summary":"Engineer.","competencies":[` +
		`{"name":"Go","level":4,"evidence_quote":"built services in Go","source_span":"line 3"},` +
		`{"name":"Rust","level":5,"evidence_quote":"   ","source_span":""}]}`
	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: mixed}, nil)
	candidates.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	out, err := profilesapp.NewProfileBuilder(candidates, profiles, llm).
		CreateFromCV(context.Background(), cid, "I built services in Go", talent.CandidateIntake{})
	require.NoError(t, err)
	require.Len(t, out.Competencies, 1, "the unevidenced 'Rust' competency is dropped")
	assert.Equal(t, "Go", out.Competencies[0].Name)
	assert.NotEmpty(t, out.Competencies[0].EvidenceQuote, "every surviving competency carries CV evidence")
}

func TestCreateFromCVRejectsEmptyCV(t *testing.T) {
	ctrl := gomock.NewController(t)
	b := profilesapp.NewProfileBuilder(mocks.NewMockCandidateRepository(ctrl), mocks.NewMockTalentProfileRepository(ctrl), mocks.NewMockLLMClient(ctrl))
	_, err := b.CreateFromCV(context.Background(), kernel.NewID(), "   ", talent.CandidateIntake{})
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestCreateFromCVUnknownCandidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	candidates.EXPECT().ByID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("nope"))
	b := profilesapp.NewProfileBuilder(candidates, mocks.NewMockTalentProfileRepository(ctrl), mocks.NewMockLLMClient(ctrl))
	_, err := b.CreateFromCV(context.Background(), kernel.NewID(), "a cv", talent.CandidateIntake{})
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
}
