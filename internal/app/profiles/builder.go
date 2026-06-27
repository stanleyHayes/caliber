// Package profiles holds the Talent Passport use-cases (EPIC-06): turning a CV
// (+ guided intake) into a structured, evidence-linked talent profile. Every
// competency must carry evidence; the model is told never to invent skills.
package profiles

import (
	"context"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
	"github.com/xcreativs/caliber/internal/domain/guard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// ProfileBuilder builds and reads a candidate's talent profile.
type ProfileBuilder struct {
	candidates talent.CandidateRepository
	profiles   talent.TalentProfileRepository
	llm        app.LLMClient
}

// NewProfileBuilder wires the use-case.
func NewProfileBuilder(candidates talent.CandidateRepository, profiles talent.TalentProfileRepository, llm app.LLMClient) *ProfileBuilder {
	return &ProfileBuilder{candidates: candidates, profiles: profiles, llm: llm}
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
	comps := make([]talent.ProfileCompetency, 0, len(parsed.Competencies))
	for _, c := range parsed.Competencies {
		// No-fabrication at the extraction boundary (CAL-044): only admit a
		// competency the model backed with a CV evidence quote. An unevidenced
		// skill is dropped — never added to the verified profile — so every
		// competency in a Talent Passport traces to a real span of the CV.
		if strings.TrimSpace(c.EvidenceQuote) == "" {
			continue
		}
		comps = append(comps, talent.ProfileCompetency{Name: c.Name, Level: c.Level, EvidenceQuote: c.EvidenceQuote, SourceSpan: c.SourceSpan})
	}
	fresh, err := talent.NewTalentProfile(candidateID, parsed.Summary, comps) // validates competency names + levels
	if err != nil {
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
