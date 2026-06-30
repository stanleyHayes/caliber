package seed

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	llmadapter "github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// ghanaianWhitelist holds curated sets of Ghana / West-Africa-plausible seed
// values. They act as a regression fence: any new seed entry must be added here
// explicitly, which forces a human plausibility check and keeps the demo pool
// locally credible.
type ghanaianWhitelist struct {
	firstNames    map[string]struct{}
	lastNames     map[string]struct{}
	locations     map[string]struct{}
	universities  map[string]struct{}
	companies     map[string]struct{}
	genericTokens []string
}

func newGhanaianWhitelist() ghanaianWhitelist {
	return ghanaianWhitelist{
		firstNames: map[string]struct{}{
			"Ama": {}, "Kofi": {}, "Esi": {}, "Yaw": {}, "Abena": {}, "Kwame": {},
			"Adwoa": {}, "Kojo": {}, "Akua": {}, "Kwesi": {}, "Afia": {}, "Kwabena": {},
			"Efua": {}, "Yaa": {}, "Kwaku": {}, "Akosua": {}, "Ebenezer": {}, "Mawuli": {},
			"Nana": {}, "Naa": {},
		},
		lastNames: map[string]struct{}{
			"Mensah": {}, "Asante": {}, "Owusu": {}, "Boateng": {}, "Sarpong": {},
			"Boadu": {}, "Agyeman": {}, "Antwi": {}, "Addo": {}, "Osei": {},
			"Adu": {}, "Tetteh": {}, "Ankrah": {}, "Darko": {}, "Appiah": {},
			"Adjei": {}, "Lartey": {}, "Quaye": {}, "Doe": {}, "Nkrumah": {},
		},
		locations: map[string]struct{}{
			locAccra: {}, locKumasi: {}, locRemote: {},
			"Takoradi": {}, "Tamale": {}, "Cape Coast": {}, "Tema": {}, "Ho": {},
			"Sunyani": {}, "Koforidua": {}, "Wa": {},
		},
		universities: map[string]struct{}{
			"Ashesi University": {},
			"Kwame Nkrumah University of Science and Technology": {},
			"University of Ghana": {},
			"University of Cape Coast": {},
			"Academic City College": {},
			"Lancaster University Ghana": {},
			"Ghana Communication Technology University": {},
			"Valley View University": {},
			"Central University": {},
			"Ghana Institute of Management and Public Administration": {},
			"Koforidua Technical University": {},
			"University of Mines and Technology": {},
		},
		companies: map[string]struct{}{
			"Hubtel": {}, "mPharma": {}, "MTN Ghana": {}, "Andela Ghana": {},
			"Fidelity Bank Ghana": {}, "Tullow Oil Ghana": {},
			"Standard Chartered Bank Ghana": {}, "GCB Bank": {}, "Ecobank Ghana": {},
			"Paystack Ghana": {}, "Flutterwave Ghana": {}, "Wave Ghana": {},
			"AirtelTigo": {}, "BCX Ghana": {}, "Vodafone Ghana": {}, "Zeepay Ghana": {},
			"Absa Bank Ghana": {}, "CAL Bank": {}, "Republic Bank Ghana": {},
			"Guaranty Trust Bank Ghana": {},
		},
		genericTokens: []string{
			"Example", "Test", "Fake", "Mock", "Placeholder", "Acme", "Generic",
			"Unknown", "Sample",
		},
	}
}

func noGenericTokens(t *testing.T, wl ghanaianWhitelist, label, value string) {
	t.Helper()
	for _, tok := range wl.genericTokens {
		assert.NotContains(t, value, tok, "%s %q contains generic token %q", label, value, tok)
	}
}

func TestGeneratorTemplates_UsePlausibleGhanaValues(t *testing.T) {
	tmpl := generatorTemplates()
	wl := newGhanaianWhitelist()

	for _, n := range tmpl.firstNames {
		_, ok := wl.firstNames[n]
		assert.Truef(t, ok, "first name %q is not in the Ghana-plausible whitelist", n)
		noGenericTokens(t, wl, "first name", n)
	}

	for _, n := range tmpl.lastNames {
		_, ok := wl.lastNames[n]
		assert.Truef(t, ok, "last name %q is not in the Ghana-plausible whitelist", n)
		noGenericTokens(t, wl, "last name", n)
	}

	for _, loc := range tmpl.locations {
		_, ok := wl.locations[loc]
		assert.Truef(t, ok, "location %q is not in the Ghana-plausible whitelist", loc)
		noGenericTokens(t, wl, "location", loc)
	}

	for _, uni := range tmpl.universities {
		_, ok := wl.universities[uni]
		assert.Truef(t, ok, "university %q is not in the Ghana-plausible whitelist", uni)
		noGenericTokens(t, wl, "university", uni)
	}

	for _, co := range tmpl.companies {
		_, ok := wl.companies[co]
		assert.Truef(t, ok, "company %q is not in the Ghana-plausible whitelist", co)
		noGenericTokens(t, wl, "company", co)
	}
}

func TestGeneratorRoles_AreGhanaPlausible(t *testing.T) {
	tmpl := generatorTemplates()
	wl := newGhanaianWhitelist()

	for _, rt := range tmpl.roles {
		_, ok := wl.locations[rt.location]
		assert.Truef(t, ok, "role %q location %q is not Ghana-plausible", rt.title, rt.location)
		assert.NotEmpty(t, rt.title, "role has a title")
		noGenericTokens(t, wl, "role title", rt.title)
	}
}

func TestGeneratorCandidateInputs_AreGhanaPlausible(t *testing.T) {
	tmpl := generatorTemplates()
	wl := newGhanaianWhitelist()

	for i := range candidateCount {
		in := buildCandidateInputs(tmpl, i)

		parts := strings.Split(in.name, " ")
		require.Len(t, parts, 2, "generated candidate name %q should be first + last", in.name)
		_, firstOK := wl.firstNames[parts[0]]
		_, lastOK := wl.lastNames[parts[1]]
		assert.Truef(t, firstOK, "candidate %d first name %q is not Ghana-plausible", i, parts[0])
		assert.Truef(t, lastOK, "candidate %d last name %q is not Ghana-plausible", i, parts[1])

		_, locOK := wl.locations[in.location]
		assert.Truef(t, locOK, "candidate %d location %q is not Ghana-plausible", i, in.location)

		uni := tmpl.universities[i%len(tmpl.universities)]
		_, uniOK := wl.universities[uni]
		assert.Truef(t, uniOK, "candidate %d university %q is not Ghana-plausible", i, uni)

		co1 := tmpl.companies[i%len(tmpl.companies)]
		co2 := tmpl.companies[(i+1)%len(tmpl.companies)]
		_, co1OK := wl.companies[co1]
		_, co2OK := wl.companies[co2]
		assert.Truef(t, co1OK, "candidate %d company1 %q is not Ghana-plausible", i, co1)
		assert.Truef(t, co2OK, "candidate %d company2 %q is not Ghana-plausible", i, co2)

		// The CV text itself must not drift into generic placeholder language.
		noGenericTokens(t, wl, "cv text", in.cv)
	}
}

func TestGenerator_GeneratedCandidates_AreGhanaPlausible(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	gen := NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)

	users := memory.NewUserRepo()
	cands := memory.NewCandidateRepo()
	profs := memory.NewTalentProfileRepo()
	roles := memory.NewRoleRepo()
	repos := Repositories{Users: users, Candidates: cands, Profiles: profs, Roles: roles}

	_, err := gen.Generate(ctx, repos)
	require.NoError(t, err)

	candidates, total, err := cands.List(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	require.Positive(t, total)

	wl := newGhanaianWhitelist()
	for _, c := range candidates {
		u, uerr := users.ByID(ctx, c.UserID)
		require.NoErrorf(t, uerr, "lookup user for candidate %s", c.ID)

		parts := strings.Split(u.Name, " ")
		require.Len(t, parts, 2, "candidate user name %q should be first + last", u.Name)
		_, firstOK := wl.firstNames[parts[0]]
		_, lastOK := wl.lastNames[parts[1]]
		assert.Truef(t, firstOK, "generated candidate first name %q is not Ghana-plausible", parts[0])
		assert.Truef(t, lastOK, "generated candidate last name %q is not Ghana-plausible", parts[1])

		_, locOK := wl.locations[c.Intake.Location]
		assert.Truef(t, locOK, "generated candidate location %q is not Ghana-plausible", c.Intake.Location)
	}
}

func TestGenerator_GeneratedEmployers_AreGhanaPlausible(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	hasher := authadapter.NewArgon2idHasher()
	gen := NewGenerator(hasher, llmadapter.NewDev(), now)

	users := memory.NewUserRepo()
	pwHash, err := hasher.Hash(DefaultPassword)
	require.NoError(t, err)

	employers, err := gen.generateEmployers(ctx, users, pwHash)
	require.NoError(t, err)
	require.NotEmpty(t, employers)

	wl := newGhanaianWhitelist()
	for _, u := range employers {
		_, ok := wl.companies[u.Name]
		assert.Truef(t, ok, "generated employer %q is not a Ghana-plausible company", u.Name)
		noGenericTokens(t, wl, "employer name", u.Name)
	}
}
