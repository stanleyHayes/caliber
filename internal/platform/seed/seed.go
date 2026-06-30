// Package seed loads a deterministic, internally-consistent demo dataset so the
// platform is demo-able out of the box (Talent Radar, two-way alerts, the
// candidate pool) and so integration tests share realistic fixtures (CAL-016).
// It builds entities only through the domain constructors, honouring every
// invariant (e.g. the candidate.ID == user.ID provisioning convention).
package seed

import (
	"context"
	"time"

	"github.com/xcreativs/caliber/internal/app"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// Demo work locations (named to keep the content table tidy and lint-clean).
const (
	locAccra  = "Accra"
	locRemote = "Remote"
	locKumasi = "Kumasi"
)

// DefaultPassword is the shared plaintext credential for every seeded demo
// account, so a reviewer can log in as any demo user during a walkthrough.
const DefaultPassword = "Demo-Caliber-2026"

// Hasher hashes a plaintext password (satisfied by the Argon2id auth adapter).
type Hasher interface {
	Hash(plain string) (string, error)
}

// Repositories is the set of repositories a demo dataset is loaded into.
type Repositories struct {
	Users      identity.UserRepository
	Candidates talent.CandidateRepository
	Profiles   talent.TalentProfileRepository
	Roles      role.RoleRepository
	Interviews interviewdom.InterviewRepository
}

// Result summarises what was loaded.
type Result struct {
	Employers  int
	Roles      int
	Candidates int
	Interviews int // pre-run screening interviews completed (CAL-101)
}

// LoadOption customises the hand-curated seed load.
type LoadOption func(*loadConfig)

type loadConfig struct {
	preRunLLM app.LLMClient
}

// WithPreRunInterviews runs screening interviews for a curated subset of seeded
// candidates during load, producing stored report cards (CAL-101).
func WithPreRunInterviews(llm app.LLMClient) LoadOption {
	return func(c *loadConfig) { c.preRunLLM = llm }
}

// Load materialises the demo dataset into repos. All demo users share
// DefaultPassword (hashed once via hasher); now stamps creation times.
func Load(ctx context.Context, repos Repositories, hasher Hasher, now time.Time, opts ...LoadOption) (Result, error) {
	cfg := &loadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	pwHash, err := hasher.Hash(DefaultPassword)
	if err != nil {
		return Result{}, err
	}
	data := demoData()

	employers, err := seedEmployers(ctx, repos, data.employers, pwHash, now)
	if err != nil {
		return Result{}, err
	}
	if err := seedRoles(ctx, repos, data.roles, employers, now); err != nil {
		return Result{}, err
	}
	if err := seedCandidates(ctx, repos, data.candidates, pwHash, now); err != nil {
		return Result{}, err
	}

	res := Result{Employers: len(data.employers), Roles: len(data.roles), Candidates: len(data.candidates)}
	preRunCount, err := maybePreRunInterviews(ctx, repos, cfg)
	if err != nil {
		return Result{}, err
	}
	res.Interviews = preRunCount
	return res, nil
}

func seedEmployers(
	ctx context.Context, repos Repositories, specs []employerSpec, pwHash string, now time.Time,
) ([]*identity.User, error) {
	employers := make([]*identity.User, len(specs))
	for i, e := range specs {
		u, err := newUser(e.email, e.name, identity.RoleEmployer, pwHash, now)
		if err != nil {
			return nil, err
		}
		if err := repos.Users.Create(ctx, u); err != nil {
			return nil, err
		}
		employers[i] = u
	}
	return employers, nil
}

func seedRoles(
	ctx context.Context, repos Repositories, specs []roleSpec, employers []*identity.User, now time.Time,
) error {
	for _, r := range specs {
		rl, err := role.NewRole(employers[r.employer].ID, r.spec(), r.rubric(), now)
		if err != nil {
			return err
		}
		if err := repos.Roles.Create(ctx, rl); err != nil {
			return err
		}
	}
	return nil
}

func seedCandidates(
	ctx context.Context, repos Repositories, specs []candidateSpec, pwHash string, now time.Time,
) error {
	for _, c := range specs {
		if err := loadCandidate(ctx, repos, c, pwHash, now); err != nil {
			return err
		}
	}
	return nil
}

func maybePreRunInterviews(ctx context.Context, repos Repositories, cfg *loadConfig) (int, error) {
	if cfg.preRunLLM == nil {
		return 0, nil
	}
	preRun, err := preRunInterviews(ctx, repos, cfg.preRunLLM, handCuratedPreRunTargets())
	if err != nil {
		return 0, err
	}
	return preRun.InterviewCount, nil
}

func loadCandidate(ctx context.Context, repos Repositories, c candidateSpec, pwHash string, now time.Time) error {
	u, err := newUser(c.email, c.name, identity.RoleCandidate, pwHash, now)
	if err != nil {
		return err
	}
	if cerr := repos.Users.Create(ctx, u); cerr != nil {
		return cerr
	}
	cand, err := talent.NewCandidate(u.ID, c.location, c.intake())
	if err != nil {
		return err
	}
	cand.ID = u.ID // provisioning convention: candidate.ID == user.ID
	if cerr := repos.Candidates.Create(ctx, cand); cerr != nil {
		return cerr
	}
	profile, err := talent.NewTalentProfile(cand.ID, c.summary, c.competencies)
	if err != nil {
		return err
	}
	profile.MarkScreened() // demo profiles are screened (visible on the Radar)
	return repos.Profiles.Create(ctx, profile)
}

func newUser(email, name string, r identity.Role, pwHash string, now time.Time) (*identity.User, error) {
	addr, err := identity.NewEmail(email)
	if err != nil {
		return nil, err
	}
	return identity.NewUser(addr, r, name, pwHash, now)
}

// --- demo content (Ghana tech ecosystem; designed to produce strong two-way
// matches so the Radar alert feed is populated) ---

type employerSpec struct {
	name  string
	email string
}

type roleSpec struct {
	employer     int // index into employers
	title        string
	location     string
	availability string
	seniority    role.Seniority
	competencies []role.Competency
	mustHaves    []string
	band         kernel.SalaryBand
}

func (r roleSpec) spec() role.RoleSpec {
	return role.RoleSpec{
		Title:            r.title,
		Location:         r.location,
		Seniority:        r.seniority,
		Availability:     r.availability,
		Responsibilities: []string{"Deliver and operate production services.", "Collaborate across the team."},
		MustHaves:        r.mustHaves,
		SalaryBand:       r.band,
	}
}

func (r roleSpec) rubric() role.Rubric {
	return role.Rubric{Competencies: r.competencies}.Normalize()
}

type candidateSpec struct {
	name         string
	email        string
	location     string
	summary      string
	targetTitles []string
	salaryFloor  float64
	competencies []talent.ProfileCompetency
}

func (c candidateSpec) intake() talent.CandidateIntake {
	return talent.CandidateIntake{
		TargetTitles:   c.targetTitles,
		Location:       c.location,
		SalaryFloor:    c.salaryFloor,
		SalaryCurrency: "GHS",
	}
}

type dataset struct {
	employers  []employerSpec
	roles      []roleSpec
	candidates []candidateSpec
}

func ghs(low, high float64) kernel.SalaryBand {
	return kernel.SalaryBand{Currency: "GHS", Low: low, High: high}
}

func comp(name string, level float64, evidence string) talent.ProfileCompetency {
	return talent.ProfileCompetency{Name: name, Level: level, EvidenceQuote: evidence, SourceSpan: "CV"}
}

func must(name string, weight float64) role.Competency {
	return role.Competency{Name: name, Weight: weight, MustHave: true}
}

func nice(name string, weight float64) role.Competency {
	return role.Competency{Name: name, Weight: weight, MustHave: false}
}

//nolint:funlen // a flat, readable demo-content table; not logic.
func demoData() dataset {
	return dataset{
		employers: []employerSpec{
			{"MTN Ghana", "talent@mtn.com.gh"},
			{"Hubtel", "talent@hubtel.com"},
			{"mPharma", "talent@mpharma.com"},
		},
		roles: []roleSpec{
			{
				employer: 0, title: "Senior Backend Engineer", location: locAccra, availability: "within 1 month",
				seniority: role.SenioritySenior, mustHaves: []string{"Go", "SQL"}, band: ghs(12000, 20000),
				competencies: []role.Competency{must("Go", 0.4), must("SQL", 0.3), nice("System design", 0.3)},
			},
			{
				employer: 1, title: "Data Engineer", location: locRemote, availability: "remote, within 2 months",
				seniority: role.SeniorityMid, mustHaves: []string{"Python", "SQL"}, band: ghs(9000, 16000),
				competencies: []role.Competency{must("Python", 0.4), must("SQL", 0.4), nice("Kubernetes", 0.2)},
			},
			{
				employer: 2, title: "Mobile Engineer", location: locAccra, availability: "within 1 month",
				seniority: role.SeniorityMid, mustHaves: []string{"TypeScript", "React"}, band: ghs(8000, 14000),
				competencies: []role.Competency{must("TypeScript", 0.4), must("React", 0.4), nice("Communication", 0.2)},
			},
			{
				employer: 0, title: "Platform Engineer", location: locAccra, availability: "within 3 months",
				seniority: role.SenioritySenior, mustHaves: []string{"Go", "Kubernetes"}, band: ghs(13000, 22000),
				competencies: []role.Competency{must("Go", 0.3), must("Kubernetes", 0.4), nice("AWS", 0.3)},
			},
			{
				employer: 1, title: "Junior Frontend Engineer", location: locKumasi, availability: "immediately",
				seniority: role.SeniorityJunior, mustHaves: []string{"React", "TypeScript"}, band: ghs(4000, 7000),
				competencies: []role.Competency{must("React", 0.5), must("TypeScript", 0.5)},
			},
		},
		candidates: []candidateSpec{
			{
				name: "Ama Mensah", email: "ama.mensah@example.com", location: locAccra,
				summary:      "Senior backend engineer with 8 years building distributed services.",
				targetTitles: []string{"Backend Engineer"}, salaryFloor: 11000,
				competencies: []talent.ProfileCompetency{
					comp("Go", 5, "Led a payments platform in Go"), comp("SQL", 4, "Designed Postgres schemas"),
					comp("System design", 4, "Architected multi-region services"),
				},
			},
			{
				name: "Kofi Asante", email: "kofi.asante@example.com", location: locRemote,
				summary:      "Data engineer focused on pipelines and warehousing.",
				targetTitles: []string{"Data Engineer"}, salaryFloor: 8500,
				competencies: []talent.ProfileCompetency{
					comp("Python", 5, "Built ETL pipelines"), comp("SQL", 5, "Modelled a data warehouse"),
					comp("Kubernetes", 3, "Deployed jobs on k8s"),
				},
			},
			{
				name: "Esi Owusu", email: "esi.owusu@example.com", location: locAccra,
				summary:      "Mobile and web engineer with a product mindset.",
				targetTitles: []string{"Mobile Engineer"}, salaryFloor: 7500,
				competencies: []talent.ProfileCompetency{
					comp("TypeScript", 5, "Shipped React Native apps"), comp("React", 5, "Built design systems"),
					comp("Communication", 4, "Led client demos"),
				},
			},
			{
				name: "Yaw Boateng", email: "yaw.boateng@example.com", location: locAccra,
				summary:      "Platform engineer specialising in Kubernetes and cloud.",
				targetTitles: []string{"Platform Engineer"}, salaryFloor: 12000,
				competencies: []talent.ProfileCompetency{
					comp("Go", 4, "Wrote operators in Go"), comp("Kubernetes", 5, "Ran production clusters"),
					comp("AWS", 4, "Managed multi-account AWS"),
				},
			},
			{
				name: "Abena Sarpong", email: "abena.sarpong@example.com", location: locKumasi,
				summary:      "Frontend engineer early in her career, strong fundamentals.",
				targetTitles: []string{"Frontend Engineer"}, salaryFloor: 4000,
				competencies: []talent.ProfileCompetency{
					comp("React", 4, "Built dashboards in React"), comp("TypeScript", 4, "Typed component libraries"),
				},
			},
			{
				name: "Kwame Boadu", email: "kwame.boadu@example.com", location: locAccra,
				summary:      "Backend engineer growing into senior scope.",
				targetTitles: []string{"Backend Engineer"}, salaryFloor: 7000,
				competencies: []talent.ProfileCompetency{
					comp("Go", 3, "Built internal APIs in Go"), comp("SQL", 3, "Wrote reporting queries"),
				},
			},
			{
				name: "Adwoa Agyeman", email: "adwoa.agyeman@example.com", location: locRemote,
				summary:      "Analytics engineer moving into data engineering.",
				targetTitles: []string{"Data Engineer"}, salaryFloor: 7500,
				competencies: []talent.ProfileCompetency{
					comp("Python", 4, "Automated analytics in Python"), comp("SQL", 3, "Built dbt models"),
				},
			},
			{
				name: "Kojo Antwi", email: "kojo.antwi@example.com", location: locAccra,
				summary:      "Enterprise Java engineer exploring new opportunities.",
				targetTitles: []string{"Software Engineer"}, salaryFloor: 9000,
				competencies: []talent.ProfileCompetency{
					comp("Java", 4, "Maintained Spring services"), comp("Spring", 3, "Built REST APIs"),
				},
			},
		},
	}
}
