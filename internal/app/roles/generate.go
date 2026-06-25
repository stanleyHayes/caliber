// Package roles holds Role-related application use-cases (Flow A.1).
package roles

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
)

// SpecGenerator turns a free-text hiring need into a structured, persisted Role.
type SpecGenerator struct {
	llm   app.LLMClient
	roles role.RoleRepository
	now   app.Clock
}

// NewSpecGenerator wires the use-case.
func NewSpecGenerator(llm app.LLMClient, repo role.RoleRepository, now app.Clock) *SpecGenerator {
	return &SpecGenerator{llm: llm, roles: repo, now: now}
}

// SystemPrompt instructs the model to emit a strict role-spec JSON document.
const SystemPrompt = `You convert a hiring manager's messy request into a structured role spec and a
weighted rubric. Respond ONLY with a JSON object matching the agreed schema (title, location,
seniority, availability, responsibilities[], must_haves[], nice_to_haves[],
salary_band{currency,low,high}, rubric[{name,weight,must_have}]). Rubric weights must sum to 1.0.`

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
	resp, err := g.llm.Complete(ctx, app.LLMRequest{System: SystemPrompt, Prompt: freeText, MaxTokens: 1024})
	if err != nil {
		return nil, kernel.Wrap(err, kernel.KindInternal, "roles: llm completion failed")
	}
	var parsed llmRoleSpec
	if err := json.Unmarshal([]byte(resp.Text), &parsed); err != nil {
		return nil, kernel.Wrap(err, kernel.KindInvalid, "roles: could not parse model output as role spec")
	}
	spec, rubric := toDomain(parsed)
	r, err := role.NewRole(employerID, spec, rubric, g.now())
	if err != nil {
		return nil, err
	}
	if err := g.roles.Create(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func toDomain(p llmRoleSpec) (role.RoleSpec, role.Rubric) {
	sen, err := role.ParseSeniority(p.Seniority)
	if err != nil {
		sen = role.SeniorityMid
	}
	spec := role.RoleSpec{
		Title:            p.Title,
		Location:         p.Location,
		Seniority:        sen,
		Availability:     p.Availability,
		Responsibilities: p.Responsibilities,
		MustHaves:        p.MustHaves,
		NiceToHaves:      p.NiceToHaves,
		SalaryBand:       kernel.SalaryBand{Currency: p.SalaryBand.Currency, Low: p.SalaryBand.Low, High: p.SalaryBand.High},
	}
	comps := make([]role.Competency, 0, len(p.Rubric))
	for _, c := range p.Rubric {
		comps = append(comps, role.Competency{Name: c.Name, Weight: c.Weight, MustHave: c.MustHave})
	}
	return spec, role.Rubric{Competencies: comps}.Normalize()
}
