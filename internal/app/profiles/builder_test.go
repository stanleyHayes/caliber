package profiles_test

import (
	"context"
	"errors"
	"strings"
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

func TestCreateFromCVEmbedsEvidencedProfileBeforeCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: extractJSON}, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, text string) ([]float32, error) {
		assert.Contains(t, text, "Go")
		assert.Contains(t, text, "built services in Go")
		assert.NotContains(t, text, "Senior engineer.", "profile recall embedding must not include free-text summary")
		return []float32{0.4, 0.8}, nil
	})
	candidates.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	var created *talent.TalentProfile
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *talent.TalentProfile) error { created = p; return nil })

	out, err := profilesapp.NewProfileBuilder(candidates, profiles, llm, profilesapp.WithEmbedder(embedder)).
		CreateFromCV(context.Background(), cid, "I built services in Go", talent.CandidateIntake{})
	require.NoError(t, err)
	assert.Equal(t, []float32{0.4, 0.8}, out.Embedding)
	require.NotNil(t, created)
	assert.Equal(t, []float32{0.4, 0.8}, created.Embedding)
}

func TestCreateFromCVEmbeddingErrorStopsBeforeMerge(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: extractJSON}, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return(nil, errors.New("embed down"))

	_, err = profilesapp.NewProfileBuilder(candidates, profiles, llm, profilesapp.WithEmbedder(embedder)).
		CreateFromCV(context.Background(), cid, "I built services in Go", talent.CandidateIntake{Location: "Accra"})
	require.Error(t, err)
}

func TestCreateFromCVRejectsOversizedText(t *testing.T) {
	ctrl := gomock.NewController(t)
	// The length guard runs before any repo or LLM call, so the mocks expect
	// nothing: an abusive payload is rejected before it can drive token cost.
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)

	huge := strings.Repeat("a", 200_001) // one rune past the 200k cap
	_, err := profilesapp.NewProfileBuilder(candidates, profiles, llm).
		CreateFromCV(context.Background(), kernel.NewID(), huge, talent.CandidateIntake{})
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
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

// TestCreateFromCVDropsFabricatedEvidence is the CAL-044 no-fabrication guard: a
// competency whose evidence quote is non-empty but does NOT appear in the CV — the
// model invented it — must be dropped, not admitted to the verified profile.
func TestCreateFromCVDropsFabricatedEvidence(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)

	// 'Go' grounds in the CV; 'Kubernetes' cites a quote the CV never contained.
	const mixed = `{"summary":"Engineer.","competencies":[` +
		`{"name":"Go","level":4,"evidence_quote":"built services in Go","source_span":"line 3"},` +
		`{"name":"Kubernetes","level":5,"evidence_quote":"ran a 500-node Kubernetes fleet","source_span":"line 9"}]}`
	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: mixed}, nil)
	candidates.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	out, err := profilesapp.NewProfileBuilder(candidates, profiles, llm).
		CreateFromCV(context.Background(), cid, "I built services in Go", talent.CandidateIntake{})
	require.NoError(t, err)
	require.Len(t, out.Competencies, 1, "the fabricated 'Kubernetes' competency is dropped")
	assert.Equal(t, "Go", out.Competencies[0].Name)
}

// TestCreateFromCVGroundsQuoteDespiteCaseAndWhitespace: a genuine quote that
// differs from the CV only in case or whitespace still grounds, so extraction
// artifacts do not cause a real competency to be dropped.
func TestCreateFromCVGroundsQuoteDespiteCaseAndWhitespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)

	const extract = `{"summary":"Lead.","competencies":[` +
		`{"name":"leadership","level":4,"evidence_quote":"led the payments platform team","source_span":"exp#1"}]}`
	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: extract}, nil)
	candidates.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(nil, kernel.NotFound("none"))
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	// CV has different case and irregular whitespace than the model's quote.
	out, err := profilesapp.NewProfileBuilder(candidates, profiles, llm).
		CreateFromCV(context.Background(), cid, "Led  the\nPayments Platform Team at Kuvora", talent.CandidateIntake{})
	require.NoError(t, err)
	require.Len(t, out.Competencies, 1, "a genuine quote grounds despite case/whitespace differences")
	assert.Equal(t, "leadership", out.Competencies[0].Name)
}

func TestCreateFromCVUpdatesExistingProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	llm := mocks.NewMockLLMClient(ctrl)
	embedder := mocks.NewMockEmbedder(ctrl)
	cid := kernel.NewID()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)
	existing, err := talent.NewTalentProfile(cid, "stale summary",
		[]talent.ProfileCompetency{{Name: "COBOL", Level: 2, EvidenceQuote: "old cv", SourceSpan: "line 1"}})
	require.NoError(t, err)

	candidates.EXPECT().ByID(gomock.Any(), cid).Return(cand, nil)
	llm.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(app.LLMResponse{Text: extractJSON}, nil)
	embedder.EXPECT().Embed(gomock.Any(), gomock.Any()).Return([]float32{0.6, 0.1}, nil)
	candidates.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	// A profile already exists -> the upsert takes the update branch, not Create.
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(existing, nil)
	var updated *talent.TalentProfile
	profiles.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *talent.TalentProfile) error { updated = p; return nil })

	out, err := profilesapp.NewProfileBuilder(candidates, profiles, llm, profilesapp.WithEmbedder(embedder)).
		CreateFromCV(context.Background(), cid, "I built services in Go", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Same(t, existing, out, "the existing profile record is updated in place, not replaced")
	assert.Equal(t, "Senior engineer.", out.Summary, "summary is refreshed from the new extraction")
	assert.Equal(t, []float32{0.6, 0.1}, out.Embedding, "embedding is refreshed alongside the profile")
	require.Len(t, out.Competencies, 1)
	assert.Equal(t, "Go", out.Competencies[0].Name, "competencies are replaced, the stale COBOL entry is gone")
}

func TestGetProfileReturnsProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	cid := kernel.NewID()
	prof, err := talent.NewTalentProfile(cid, "summary",
		[]talent.ProfileCompetency{{Name: "Go", Level: 4, EvidenceQuote: "built services", SourceSpan: "line 3"}})
	require.NoError(t, err)
	profiles.EXPECT().ByCandidateID(gomock.Any(), cid).Return(prof, nil)

	b := profilesapp.NewProfileBuilder(mocks.NewMockCandidateRepository(ctrl), profiles, mocks.NewMockLLMClient(ctrl))
	out, err := b.GetProfile(context.Background(), cid)
	require.NoError(t, err)
	assert.Equal(t, prof, out)
}

func TestGetProfileNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := mocks.NewMockTalentProfileRepository(ctrl)
	profiles.EXPECT().ByCandidateID(gomock.Any(), gomock.Any()).Return(nil, kernel.NotFound("none"))
	b := profilesapp.NewProfileBuilder(mocks.NewMockCandidateRepository(ctrl), profiles, mocks.NewMockLLMClient(ctrl))
	_, err := b.GetProfile(context.Background(), kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
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
