package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/app/roles"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// candidateCount is the deterministic size of the generated candidate pool. It
// sits inside the CAL-098 acceptance range of 50–60 realistic CVs/profiles.
const candidateCount = 55

// seniorityLabels are the candidate seniority tiers used to shape experience
// and salary expectations in generated CVs.
var seniorityLabels = []string{"junior", "mid", "senior", "lead"} //nolint:gochecknoglobals // fixed ordered set

// Generator produces a deterministic, Ghana/West-Africa-plausible demo dataset
// by driving the real CV parser (profiles.ProfileBuilder) and the real role-spec
// parser (roles.SpecGenerator). Every profile that survives parsing carries
// traceable CV evidence, satisfying the no-fabrication guardrail.
type Generator struct {
	hasher Hasher
	llm    app.LLMClient
	now    func() time.Time
}

// NewGenerator wires a seed generator. now is usually time.Now; it is injected
// so tests can pin creation timestamps.
func NewGenerator(hasher Hasher, llm app.LLMClient, now func() time.Time) *Generator {
	return &Generator{hasher: hasher, llm: llm, now: now}
}

// Generate runs the full pipeline: 6–8 employers, 8–12 roles, and 50–60
// candidate profiles, all produced through the production parsers.
func (g *Generator) Generate(ctx context.Context, repos Repositories) (Result, error) {
	pwHash, err := g.hasher.Hash(DefaultPassword)
	if err != nil {
		return Result{}, err
	}

	employers, err := g.generateEmployers(ctx, repos.Users, pwHash)
	if err != nil {
		return Result{}, err
	}

	roleList, err := g.generateRoles(ctx, repos.Roles, employers)
	if err != nil {
		return Result{}, err
	}

	candCount, err := g.generateCandidates(ctx, repos, pwHash)
	if err != nil {
		return Result{}, err
	}

	preRun, err := preRunInterviews(ctx, repos, g.llm, generatedPreRunTargets())
	if err != nil {
		return Result{}, fmt.Errorf("pre-run interviews: %w", err)
	}

	preSeed, err := preSeedAgentState(ctx, repos, g.llm, repos.Applications, generatedPreSeedTargets())
	if err != nil {
		return Result{}, fmt.Errorf("pre-seed agent state: %w", err)
	}

	return Result{
		Employers:    len(employers),
		Roles:        len(roleList),
		Candidates:   candCount,
		Interviews:   preRun.InterviewCount,
		Applications: preSeed.ApplicationCount,
	}, nil
}

func (g *Generator) generateEmployers(
	ctx context.Context, users identity.UserRepository, pwHash string,
) ([]*identity.User, error) {
	employerTable := []struct {
		name  string
		email string
	}{
		{"MTN Ghana", "talent@mtn.com.gh"},
		{"Hubtel", "talent@hubtel.com"},
		{"mPharma", "talent@mpharma.com"},
		{"Andela Ghana", "talent@andela.com.gh"},
		{"Standard Chartered Bank Ghana", "careers@sc.com.gh"},
		{"Fidelity Bank Ghana", "careers@fidelitybank.com.gh"},
		{"Tullow Oil Ghana", "careers@tullowoil.com.gh"},
		{"Flutterwave Ghana", "careers@flutterwave.com.gh"},
	}

	out := make([]*identity.User, 0, len(employerTable))
	for _, e := range employerTable {
		u, err := newUser(e.email, e.name, identity.RoleEmployer, pwHash, g.now())
		if err != nil {
			return nil, err
		}
		if err := users.Create(ctx, u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func (g *Generator) generateRoles(
	ctx context.Context, repo role.RoleRepository, employers []*identity.User,
) ([]*role.Role, error) {
	tmpl := generatorTemplates()
	gen := roles.NewSpecGenerator(g.llm, repo, g.now)

	out := make([]*role.Role, 0, len(tmpl.roles))
	for _, rt := range tmpl.roles {
		employer := employers[rt.employerIdx]
		freeText := buildRoleFreeText(rt)
		rl, err := gen.Generate(ctx, employer.ID, freeText)
		if err != nil {
			return nil, fmt.Errorf("generate role %q: %w", rt.title, err)
		}
		out = append(out, rl)
	}
	return out, nil
}

func buildRoleFreeText(rt roleTemplate) string {
	var b strings.Builder
	b.WriteString(rt.title)
	b.WriteString("\n")
	fmt.Fprintf(&b, "We are hiring a %s %s based in %s. ", rt.seniority.String(), rt.title, rt.location)
	fmt.Fprintf(&b, "You will %s and %s. ", rt.responsibilities[0], strings.ToLower(rt.responsibilities[1]))
	fmt.Fprintf(&b, "Must-haves: %s. ", strings.Join(rt.mustHaves, ", "))
	fmt.Fprintf(&b, "Nice-to-haves: %s. ", strings.Join(rt.niceToHaves, ", "))
	fmt.Fprintf(&b, "Compensation: %s. ", rt.compensation)
	fmt.Fprintf(&b, "Availability: %s.", rt.availability)
	return b.String()
}

func (g *Generator) generateCandidates(ctx context.Context, repos Repositories, pwHash string) (int, error) {
	tmpl := generatorTemplates()
	heroes := heroCandidateMap()
	builder := profilesapp.NewProfileBuilder(repos.Candidates, repos.Profiles, g.llm)

	for i := range candidateCount {
		inputs, ok := heroes[i]
		if !ok {
			inputs = buildCandidateInputs(tmpl, i)
		}
		candID, err := g.createCandidateAccount(ctx, repos, pwHash, inputs)
		if err != nil {
			return 0, fmt.Errorf("create candidate %d: %w", i, err)
		}
		if err := g.extractAndScreenProfile(ctx, repos, builder, candID, inputs); err != nil {
			return 0, fmt.Errorf("extract candidate %d: %w", i, err)
		}
	}
	return candidateCount, nil
}

func (g *Generator) createCandidateAccount(
	ctx context.Context, repos Repositories, pwHash string, in candidateInputs,
) (kernel.ID, error) {
	u, err := newUser(in.email, in.name, identity.RoleCandidate, pwHash, g.now())
	if err != nil {
		return "", err
	}
	if err := repos.Users.Create(ctx, u); err != nil {
		return "", err
	}

	cand, err := talent.NewCandidate(u.ID, in.location, in.intake)
	if err != nil {
		return "", err
	}
	cand.ID = u.ID
	if err := repos.Candidates.Create(ctx, cand); err != nil {
		return "", err
	}
	return cand.ID, nil
}

func (g *Generator) extractAndScreenProfile(
	ctx context.Context,
	repos Repositories,
	builder *profilesapp.ProfileBuilder,
	candidateID kernel.ID,
	in candidateInputs,
) error {
	profile, err := builder.CreateFromCV(ctx, candidateID, in.cv, in.intake)
	if err != nil {
		return err
	}

	// Profiles produced by the parser start at cv_only; mark them screened so
	// they appear on the Talent Radar and in shortlists, matching the demo
	// convention used by the hand-curated fixtures.
	profile.MarkScreened()
	if err := repos.Profiles.Update(ctx, profile); err != nil {
		return err
	}
	return nil
}

// candidateInputs bundles the deterministic inputs for a single generated
// candidate.
type candidateInputs struct {
	name       string
	email      string
	location   string
	family     familyTemplate
	seniority  string
	years      int
	salaryFloor float64
	cv         string
	intake     talent.CandidateIntake
}

func buildCandidateInputs(
	tmpl struct {
		firstNames   []string
		lastNames    []string
		locations    []string
		universities []string
		companies    []string
		families     []familyTemplate
		roles        []roleTemplate
	},
	idx int,
) candidateInputs {
	first := tmpl.firstNames[idx%len(tmpl.firstNames)]
	last := tmpl.lastNames[(idx/len(tmpl.firstNames))%len(tmpl.lastNames)]
	name := first + " " + last
	email := strings.ToLower(fmt.Sprintf("%s.%s.gen%d@example.com", first, last, idx))
	location := tmpl.locations[idx%len(tmpl.locations)]
	family := tmpl.families[idx%len(tmpl.families)]
	seniority := seniorityLabels[idx%len(seniorityLabels)]
	years := yearsFor(seniority, idx)
	salaryFloor := salaryFloorFor(seniority, idx)

	intake := talent.CandidateIntake{
		TargetTitles:   family.targetTitles,
		Location:       location,
		SalaryFloor:    salaryFloor,
		SalaryCurrency: "GHS",
	}

	cv := buildCV(cvInputs{
		name:       name,
		family:     family,
		seniority:  seniority,
		years:      years,
		location:   location,
		university: tmpl.universities[idx%len(tmpl.universities)],
		gradYear:   2008 + idx%12,
		company1:   tmpl.companies[idx%len(tmpl.companies)],
		company2:   tmpl.companies[(idx+1)%len(tmpl.companies)],
	})

	return candidateInputs{
		name:        name,
		email:       email,
		location:    location,
		family:      family,
		seniority:   seniority,
		years:       years,
		salaryFloor: salaryFloor,
		cv:          cv,
		intake:      intake,
	}
}

// cvInputs bundles the fields needed to render a synthetic CV.
type cvInputs struct {
	name       string
	family     familyTemplate
	seniority  string
	years      int
	location   string
	university string
	gradYear   int
	company1   string
	company2   string
}

func buildCV(in cvInputs) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", in.name)
	fmt.Fprintf(&b, "%s\n", in.location)
	fmt.Fprintf(&b, "Software professional focused on %s\n\n", in.family.name)

	fmt.Fprintln(&b, "SUMMARY")
	fmt.Fprintf(&b, "%s %s with %d years of experience in %s. ",
		titleCase(in.seniority), in.family.name, in.years, in.family.name)
	fmt.Fprintf(&b, "Skilled in %s. Experienced with %s.\n\n",
		strings.Join(in.family.skills, ", "), strings.Join(in.family.tools, ", "))

	fmt.Fprintln(&b, "EXPERIENCE")
	fmt.Fprintf(&b, "- %s, %s, 2020–2024: Delivered production features using %s and %s.\n",
		jobTitleFor(in.family.name, in.seniority), in.company1,
		strings.Join(in.family.skills, ", "), strings.Join(in.family.tools, ", "))
	fmt.Fprintf(&b, "- %s, %s, 2018–2020: Built and maintained systems with %s, gaining hands-on experience across the stack.\n\n",
		jobTitleFor(in.family.name, "mid"), in.company2, strings.Join(in.family.skills, ", "))

	fmt.Fprintln(&b, "EDUCATION")
	fmt.Fprintf(&b, "- %s, %s, %d\n\n", in.family.degree, in.university, in.gradYear)

	fmt.Fprintln(&b, "SKILLS")
	allSkills := append([]string(nil), in.family.skills...)
	allSkills = append(allSkills, in.family.tools...)
	b.WriteString(strings.Join(allSkills, ", "))
	b.WriteByte('\n')

	return b.String()
}

func jobTitleFor(familyName, seniority string) string {
	base := strings.TrimSuffix(familyName, "ing")
	if familyName == "Product Management" {
		base = "Product Manag"
	}
	return titleCase(seniority) + " " + base + "er"
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func yearsFor(seniority string, idx int) int {
	switch seniority {
	case "junior":
		return 1 + idx%3
	case "mid":
		return 3 + idx%4
	case "senior":
		return 6 + idx%5
	case "lead":
		return 10 + idx%6
	default:
		return 3
	}
}

func salaryFloorFor(seniority string, idx int) float64 {
	switch seniority {
	case "junior":
		return 3500 + float64(idx%3)*500
	case "mid":
		return 7000 + float64(idx%4)*1000
	case "senior":
		return 12000 + float64(idx%5)*1000
	case "lead":
		return 18000 + float64(idx%4)*2000
	default:
		return 7000
	}
}
