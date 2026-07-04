// Package profiles holds the Talent Passport use-cases (EPIC-06): turning a CV
// (+ guided intake) into a structured, evidence-linked talent profile. Every
// competency must carry evidence; the model is told never to invent skills.
package profiles

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
	"github.com/xcreativs/caliber/internal/domain/guard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// maxCVTextRunes bounds the CV text fed to the extraction model. A real CV is a
// few pages (tens of KB); this generous ceiling rejects an abusive payload before
// it inflates LLM token cost and latency (a cost-amplification DoS), independent
// of which inbound path supplied the text (CAL-120). Runes, not bytes, so the
// limit is consistent across scripts.
const maxCVTextRunes = 200_000

// ProfileBuilder builds and reads a candidate's talent profile.
type ProfileBuilder struct {
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	llm        app.LLMClient
	embedder   app.Embedder
}

// BuilderOption configures the profile builder.
type BuilderOption func(*ProfileBuilder)

// WithEmbedder enables profile embedding before persistence.
func WithEmbedder(embedder app.Embedder) BuilderOption {
	return func(b *ProfileBuilder) { b.embedder = embedder }
}

// NewProfileBuilder wires the use-case.
func NewProfileBuilder(
	candidates talent.CandidateRepository,
	profiles talent.TalentProfileRepository,
	llm app.LLMClient,
	opts ...BuilderOption,
) *ProfileBuilder {
	b := &ProfileBuilder{candidates: candidates, profiles: profiles, llm: llm}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

type llmProfile struct {
	Summary      string `json:"summary"`
	Competencies []struct {
		Name          string  `json:"name"`
		Level         float64 `json:"level"`
		EvidenceQuote string  `json:"evidence_quote"`
		SourceSpan    string  `json:"source_span"`
	} `json:"competencies"`
}

// CreateFromCV extracts an evidence-linked profile from a CV, merges the guided
// intake into the candidate, and upserts the profile. The candidate must exist.
func (b *ProfileBuilder) CreateFromCV(
	ctx context.Context, candidateID kernel.ID, cvText string, intake talent.CandidateIntake,
) (*talent.TalentProfile, error) {
	if strings.TrimSpace(cvText) == "" {
		return nil, kernel.Invalid("talent: cv text is required")
	}
	if utf8.RuneCountInString(cvText) > maxCVTextRunes {
		return nil, kernel.Invalidf("talent: cv text exceeds the %d character limit", maxCVTextRunes)
	}
	cand, err := b.candidates.ByID(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	parsed, err := app.DecodeJSON[llmProfile](ctx, b.llm,
		prompts.Get(prompts.IDCVExtract).Request(guard.FenceUntrusted("CANDIDATE_CV", cvText)),
		app.DefaultLLMAttempts, "talent: profile extraction")
	if err != nil {
		return nil, err
	}
	comps := groundedCompetencies(cvText, parsed)
	fresh, err := talent.NewTalentProfile(candidateID, parsed.Summary, comps) // validates competency names + levels
	if err != nil {
		return nil, err
	}
	if err := b.embed(ctx, fresh); err != nil {
		return nil, err
	}

	if err := b.mergeIntake(ctx, cand, intake); err != nil {
		return nil, err
	}
	return b.upsert(ctx, candidateID, fresh)
}

// GetProfile returns a candidate's talent profile.
func (b *ProfileBuilder) GetProfile(ctx context.Context, candidateID kernel.ID) (*talent.TalentProfile, error) {
	return b.profiles.ByCandidateID(ctx, candidateID)
}

func (b *ProfileBuilder) mergeIntake(ctx context.Context, cand *talent.Candidate, intake talent.CandidateIntake) error {
	cand.Intake = intake
	if strings.TrimSpace(intake.Location) != "" {
		cand.Location = intake.Location
	}
	return b.candidates.Update(ctx, cand)
}

func (b *ProfileBuilder) upsert(ctx context.Context, candidateID kernel.ID, fresh *talent.TalentProfile) (*talent.TalentProfile, error) {
	if existing, err := b.profiles.ByCandidateID(ctx, candidateID); err == nil {
		existing.Summary = fresh.Summary
		existing.Competencies = fresh.Competencies
		existing.Embedding = cloneEmbedding(fresh.Embedding)
		if uerr := b.profiles.Update(ctx, existing); uerr != nil {
			return nil, uerr
		}
		return existing, nil
	}
	if err := b.profiles.Create(ctx, fresh); err != nil {
		return nil, err
	}
	return fresh, nil
}

func (b *ProfileBuilder) embed(ctx context.Context, profile *talent.TalentProfile) error {
	if b.embedder == nil {
		return nil
	}
	text := profile.EmbeddingText()
	if strings.TrimSpace(text) == "" {
		return nil
	}
	embedding, err := b.embedder.Embed(ctx, text)
	if err != nil {
		return err
	}
	profile.Embedding = cloneEmbedding(embedding)
	return nil
}

// groundedCompetencies keeps only the competencies whose evidence quote actually
// appears in the CV (CAL-044 no-fabrication). A merely non-empty quote is not
// enough — the model could invent one — so the quote must ground to a real span
// of the source (matched case-insensitively and whitespace-normalized to tolerate
// extraction artifacts). An unevidenced or fabricated competency is dropped, never
// added to the verified profile: a hard guardrail prefers a lost skill to an
// invented one.
func groundedCompetencies(cvText string, parsed llmProfile) []talent.ProfileCompetency {
	normalizedCV := normalizeForMatch(cvText)
	comps := make([]talent.ProfileCompetency, 0, len(parsed.Competencies))
	for _, c := range parsed.Competencies {
		quote := normalizeForMatch(c.EvidenceQuote)
		if quote == "" || !strings.Contains(normalizedCV, quote) {
			continue
		}
		comps = append(comps, talent.ProfileCompetency{Name: c.Name, Level: c.Level, EvidenceQuote: c.EvidenceQuote, SourceSpan: c.SourceSpan})
	}
	return comps
}

// normalizeForMatch lower-cases s and collapses every run of whitespace to a
// single space (trimming the ends), so an evidence quote grounds against the CV
// despite case and whitespace/formatting differences introduced by extraction.
func normalizeForMatch(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func cloneEmbedding(v []float32) []float32 {
	if len(v) == 0 {
		return nil
	}
	out := make([]float32, len(v))
	copy(out, v)
	return out
}
