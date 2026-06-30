package seed_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authadapter "github.com/xcreativs/caliber/internal/adapters/outbound/auth"
	"github.com/xcreativs/caliber/internal/adapters/outbound/embeddings"
	llmadapter "github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	matchingapp "github.com/xcreativs/caliber/internal/app/matching"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/platform/seed"
)

// heroEmail returns the deterministic, normalized email for a hero candidate slot.
func heroEmail(first, last string) string {
	return strings.ToLower(first + "." + last + ".hero@example.com")
}

func TestGenerator_HeroCandidatesExist(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)
	repos, h := newRepos()

	_, err := gen.Generate(ctx, repos)
	require.NoError(t, err)

	for _, tc := range []struct {
		email    string
		name     string
		location string
	}{
		{heroEmail("Ama", "Mensah"), "Ama Mensah", "Accra"},
		{heroEmail("Kofi", "Asante"), "Kofi Asante", "Accra"},
		{heroEmail("Esi", "Owusu"), "Esi Owusu", "Accra"},
	} {
		u, err := h.users.ByEmail(ctx, identity.Email(tc.email))
		require.NoErrorf(t, err, "hero candidate %s should exist", tc.email)
		assert.Equal(t, tc.name, u.Name)

		cand, err := h.cands.ByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, tc.location, cand.Intake.Location)

		profile, err := h.profs.ByCandidateID(ctx, cand.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, profile.Competencies)
	}
}

func TestGenerator_HeroCandidatesHaveExcellentProfiles(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)
	repos, h := newRepos()

	_, err := gen.Generate(ctx, repos)
	require.NoError(t, err)

	for _, email := range []string{
		heroEmail("Ama", "Mensah"),
		heroEmail("Kofi", "Asante"),
		heroEmail("Esi", "Owusu"),
	} {
		u, err := h.users.ByEmail(ctx, identity.Email(email))
		require.NoError(t, err)

		profile, err := h.profs.ByCandidateID(ctx, u.ID)
		require.NoError(t, err)

		compNames := make(map[string]float64, len(profile.Competencies))
		for _, c := range profile.Competencies {
			assert.NotEmpty(t, c.EvidenceQuote, "hero competency %q has CV evidence", c.Name)
			assert.NotEmpty(t, c.SourceSpan, "hero competency %q has a source span", c.Name)
			compNames[c.Name] = c.Level
		}

		assert.Contains(t, compNames, "Core skills", "hero covers the dev rubric must-have")
		assert.Contains(t, compNames, "Communication", "hero covers the Communication rubric item")
		assert.Contains(t, compNames, "System design", "hero covers the System design rubric item")
	}
}

func TestGenerator_HeroPairsProduceStrongShortlistMatches(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)
	repos, h := newRepos()

	_, err := gen.Generate(ctx, repos)
	require.NoError(t, err)

	shortlister := matchingapp.NewShortlister(
		h.roles, h.cands, h.profs, memory.NewRecaller(h.cands), embeddings.NewDev(), llmadapter.NewDev(), memory.NewMatchRepo())

	for _, tc := range []struct {
		roleTitle string
		empEmail  string
		heroEmail string
	}{
		{"Senior Backend Engineer", "talent@mtn.com.gh", heroEmail("Ama", "Mensah")},
		{"Data Engineer", "talent@hubtel.com", heroEmail("Kofi", "Asante")},
		{"Platform Engineer", "talent@mtn.com.gh", heroEmail("Esi", "Owusu")},
	} {
		emp, err := h.users.ByEmail(ctx, identity.Email(tc.empEmail))
		require.NoErrorf(t, err, "employer %s should exist", tc.empEmail)

		roleID := findRoleID(ctx, t, h.roles, tc.roleTitle)
		result, err := shortlister.GenerateShortlist(ctx, roleID, emp.ID, 0)
		require.NoErrorf(t, err, "shortlist for %s", tc.roleTitle)
		require.NotEmpty(t, result.Matches, "shortlist for %s has matches", tc.roleTitle)

		heroUser, err := h.users.ByEmail(ctx, identity.Email(tc.heroEmail))
		require.NoError(t, err)

		var heroMatch *matchingdom.Match
		for _, m := range result.Matches {
			if m.CandidateID == heroUser.ID {
				heroMatch = m
				break
			}
		}
		require.NotNilf(t, heroMatch, "hero %s should appear in the %s shortlist", tc.heroEmail, tc.roleTitle)
		assert.GreaterOrEqualf(t, heroMatch.OverallScore, 0.75, "hero %s should score excellently for %s", tc.heroEmail, tc.roleTitle)

		// In the deterministic dev path all hero candidates cover the same rubric,
		// so they cluster at the top. We assert the hero is in the top tier rather
		// than a strict #1, because the demo beat is "Flow A lands" — not "this
		// exact ordering".
		topN := 3
		require.GreaterOrEqual(t, len(result.Matches), topN, "shortlist has enough matches to check the top tier")
		foundInTopTier := false
		for i := 0; i < topN && i < len(result.Matches); i++ {
			if result.Matches[i].CandidateID == heroUser.ID {
				foundInTopTier = true
				break
			}
		}
		assert.Truef(t, foundInTopTier, "hero %s should be in the top %d matches for %s", tc.heroEmail, topN, tc.roleTitle)
	}
}

func TestGenerator_HeroPairsProduceStrongTwoWayFits(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }
	gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)
	repos, h := newRepos()

	_, err := gen.Generate(ctx, repos)
	require.NoError(t, err)

	matcher := matchingapp.NewPassiveMatcher(h.roles, h.profs, h.cands)

	for _, tc := range []struct {
		heroEmail string
		roleTitle string
	}{
		{heroEmail("Ama", "Mensah"), "Senior Backend Engineer"},
		{heroEmail("Kofi", "Asante"), "Data Engineer"},
		{heroEmail("Esi", "Owusu"), "Platform Engineer"},
	} {
		heroUser, err := h.users.ByEmail(ctx, identity.Email(tc.heroEmail))
		require.NoError(t, err)

		fits, err := matcher.RolesForCandidate(ctx, heroUser.ID, 100)
		require.NoError(t, err)
		require.NotEmpty(t, fits, "hero %s should have fitting roles", tc.heroEmail)

		var heroFit *matchingapp.RoleFit
		for i := range fits {
			if fits[i].Role.Spec.Title == tc.roleTitle {
				heroFit = &fits[i]
				break
			}
		}
		require.NotNilf(t, heroFit, "hero %s should fit %s", tc.heroEmail, tc.roleTitle)
		assert.Truef(t, heroFit.Fit.MustHavesMet, "hero %s meets must-haves for %s", tc.heroEmail, tc.roleTitle)
		assert.GreaterOrEqualf(t, heroFit.Fit.Score, 0.75, "hero %s fit for %s should be strong", tc.heroEmail, tc.roleTitle)
	}
}

func TestGenerator_HeroPairsAreDeterministic(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Unix(1700000000, 0) }

	run := func() (map[string][]string, error) {
		gen := seed.NewGenerator(authadapter.NewArgon2idHasher(), llmadapter.NewDev(), now)
		repos, h := newRepos()
		if _, err := gen.Generate(ctx, repos); err != nil {
			return nil, err
		}
		out := make(map[string][]string)
		for _, email := range []string{
			heroEmail("Ama", "Mensah"),
			heroEmail("Kofi", "Asante"),
			heroEmail("Esi", "Owusu"),
		} {
			u, err := h.users.ByEmail(ctx, identity.Email(email))
			if err != nil {
				return nil, err
			}
			profile, err := h.profs.ByCandidateID(ctx, u.ID)
			if err != nil {
				return nil, err
			}
			names := make([]string, 0, len(profile.Competencies))
			for _, c := range profile.Competencies {
				names = append(names, c.Name)
			}
			out[email] = names
		}
		return out, nil
	}

	first, err := run()
	require.NoError(t, err)
	second, err := run()
	require.NoError(t, err)
	assert.Equal(t, first, second, "hero candidate profiles are deterministic across runs")
}

func findRoleID(ctx context.Context, t *testing.T, roles *memory.RoleRepo, title string) kernel.ID {
	t.Helper()
	open, _, err := roles.ListOpen(ctx, kernel.NewPage(1, 100))
	require.NoError(t, err)
	for _, rl := range open {
		if rl.Spec.Title == title {
			return rl.ID
		}
	}
	t.Fatalf("role %q not found", title)
	return ""
}
