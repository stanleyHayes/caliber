// Package roles holds Role-related application use-cases (Flow A.1).
package roles

import (
	"context"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/app/prompts"
	"github.com/xcreativs/caliber/internal/domain/guard"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// MaxFreeTextLen bounds the free-text hiring need (CAL-111): it is untrusted
// input forwarded to the model, so it is length-capped before generation to cap
// cost and resource use.
const MaxFreeTextLen = 8000

// SpecGenerator turns a free-text hiring need into a structured, persisted Role.
type SpecGenerator struct {
	llm      app.LLMClient
	roles    role.RoleRepository
	embedder app.Embedder
	now      app.Clock
}

// SpecGeneratorOption configures the role-spec generator.
type SpecGeneratorOption func(*SpecGenerator)

// WithEmbedder enables role-spec embedding before persistence.
func WithEmbedder(embedder app.Embedder) SpecGeneratorOption {
	return func(g *SpecGenerator) { g.embedder = embedder }
}

// NewSpecGenerator wires the use-case.
func NewSpecGenerator(llm app.LLMClient, repo role.RoleRepository, now app.Clock, opts ...SpecGeneratorOption) *SpecGenerator {
	g := &SpecGenerator{llm: llm, roles: repo, now: now}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

type llmRoleSpec struct {
	Title            string   `json:"title"`
	Location         string   `json:"location"`
	Seniority        string   `json:"seniority"`
	Availability     string   `json:"availability"`
	Responsibilities []string `json:"responsibilities"`
	MustHaves        []string `json:"must_haves"`
	NiceToHaves      []string `json:"nice_to_haves"`
	SalaryBand       struct {
		Currency string  `json:"currency"`
		Low      float64 `json:"low"`
		High     float64 `json:"high"`
	} `json:"salary_band"`
	Rubric []struct {
		Name     string  `json:"name"`
		Weight   float64 `json:"weight"`
		MustHave bool    `json:"must_have"`
	} `json:"rubric"`
}

// Generate produces a validated draft Role from free text and persists it.
func (g *SpecGenerator) Generate(ctx context.Context, employerID kernel.ID, freeText string) (*role.Role, error) {
	if employerID.IsZero() {
		return nil, kernel.Invalid("roles: employer id is required")
	}
	if strings.TrimSpace(freeText) == "" {
		return nil, kernel.Invalid("roles: hiring need text is required")
	}
	if len([]rune(freeText)) > MaxFreeTextLen {
		return nil, kernel.Invalidf("roles: hiring need text exceeds %d characters", MaxFreeTextLen)
	}
	parsed, err := app.DecodeJSON[llmRoleSpec](ctx, g.llm,
		prompts.Get(prompts.IDRoleSpec).Request(guard.FenceUntrusted("HIRING_NEED", freeText)),
		app.DefaultLLMAttempts, "roles: role-spec generation")
	if err != nil {
		return nil, err
	}
	spec, rubric := toDomain(parsed)
	// No salary fallback: when the hiring need states no budget the band is left
	// blank (the UI shows "Not specified" and matching skips the salary gate)
	// rather than fabricating a figure the employer never gave.
	r, err := role.NewRole(employerID, spec, rubric, g.now())
	if err != nil {
		return nil, err
	}
	if err := g.embed(ctx, r); err != nil {
		return nil, err
	}
	if err := g.roles.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (g *SpecGenerator) embed(ctx context.Context, r *role.Role) error {
	if g.embedder == nil {
		return nil
	}
	embedding, err := g.embedder.Embed(ctx, r.EmbeddingText())
	if err != nil {
		return err
	}
	r.Embedding = cloneEmbedding(embedding)
	return nil
}

func toDomain(p llmRoleSpec) (role.RoleSpec, role.Rubric) {
	sen, err := role.ParseSeniority(p.Seniority)
	if err != nil {
		sen = role.SeniorityMid
	}
	// Sanitize role-spec text at ingestion so the canonical stored names/title
	// are free of format/control runes; this keeps the prompt-visible rubric
	// names byte-identical to what is stored, so a model's competency_tag echo
	// round-trips to a real rubric name.
	spec := role.RoleSpec{
		Title:            guard.Sanitize(p.Title),
		Location:         guard.Sanitize(p.Location),
		Seniority:        sen,
		Availability:     p.Availability,
		Responsibilities: p.Responsibilities,
		MustHaves:        p.MustHaves,
		NiceToHaves:      p.NiceToHaves,
		SalaryBand:       kernel.SalaryBand{Currency: p.SalaryBand.Currency, Low: p.SalaryBand.Low, High: p.SalaryBand.High},
	}
	comps := make([]role.Competency, 0, len(p.Rubric))
	for _, c := range p.Rubric {
		comps = append(comps, role.Competency{Name: guard.Sanitize(c.Name), Weight: c.Weight, MustHave: c.MustHave})
	}
	return spec, role.Rubric{Competencies: comps}.Normalize()
}

func cloneEmbedding(v []float32) []float32 {
	if len(v) == 0 {
		return nil
	}
	out := make([]float32, len(v))
	copy(out, v)
	return out
}
