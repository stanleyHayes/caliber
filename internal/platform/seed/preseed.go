package seed

import (
	"context"
	"fmt"

	"github.com/xcreativs/caliber/internal/app"
	candidateagentapp "github.com/xcreativs/caliber/internal/app/candidateagent"
	agentdom "github.com/xcreativs/caliber/internal/domain/candidateagent"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// preSeedResult summarises how many agent applications were submitted during
// demo seeding.
type preSeedResult struct {
	ApplicationCount int
}

// preSeedTarget identifies a candidate/role pair to pre-seed with agent state.
// The role title is used for graceful skipping; the agent still scans the full
// open-role pool for the candidate and applies to every honest strong match.
type preSeedTarget struct {
	candidateEmail string
	roleTitle      string
}

// generatedPreSeedTargets selects the generated hero candidates whose profiles
// are tuned to cover strong-match roles. One candidate is intentionally left
// without pre-seeded applications so Flow C can still be run live in the demo.
func generatedPreSeedTargets() []preSeedTarget {
	return []preSeedTarget{
		{candidateEmail: "ama.mensah.hero@example.com", roleTitle: "Senior Backend Engineer"},
		{candidateEmail: "kofi.asante.hero@example.com", roleTitle: "Data Engineer"},
		{candidateEmail: "esi.owusu.hero@example.com", roleTitle: "Platform Engineer"},
	}
}

// handCuratedPreSeedTargets mirrors the hero-pair idea for the hand-curated
// demo dataset. Ama, Kofi, Esi, Yaw and Abena are pre-seeded; Kojo is left
// live for a demo-time agent run.
func handCuratedPreSeedTargets() []preSeedTarget {
	return []preSeedTarget{
		{candidateEmail: "ama.mensah@example.com", roleTitle: "Senior Backend Engineer"},
		{candidateEmail: "kofi.asante@example.com", roleTitle: "Data Engineer"},
		{candidateEmail: "esi.owusu@example.com", roleTitle: "Mobile Engineer"},
		{candidateEmail: "yaw.boateng@example.com", roleTitle: "Platform Engineer"},
		{candidateEmail: "abena.sarpong@example.com", roleTitle: "Junior Frontend Engineer"},
	}
}

// maybePreSeedAgentState runs the candidate agent for the configured targets
// when both an LLM and an application repository are supplied.
func maybePreSeedAgentState(ctx context.Context, repos Repositories, cfg *loadConfig) (int, error) {
	if cfg.preSeedLLM == nil || cfg.preSeedApps == nil {
		return 0, nil
	}
	preSeed, err := preSeedAgentState(ctx, repos, cfg.preSeedLLM, cfg.preSeedApps, handCuratedPreSeedTargets())
	if err != nil {
		return 0, err
	}
	return preSeed.ApplicationCount, nil
}

// preSeedAgentState runs the autonomous candidate agent for each supplied
// target, storing honest, grounded applications so the wake-up view is crisp
// without requiring a live run first (CAL-102).
func preSeedAgentState(
	ctx context.Context,
	repos Repositories,
	llm app.LLMClient,
	apps agentdom.ApplicationRepository,
	targets []preSeedTarget,
) (preSeedResult, error) {
	runner := candidateagentapp.NewAgentRunner(
		repos.Candidates,
		repos.Profiles,
		repos.Roles,
		apps,
		llm,
		candidateagentapp.WithWakeUpInsights(repos.Interviews, nil),
	)

	roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 1000))
	if err != nil {
		return preSeedResult{}, fmt.Errorf("seed: list roles for pre-seed: %w", err)
	}
	roleByTitle := make(map[string]kernel.ID, len(roles))
	for _, rl := range roles {
		roleByTitle[rl.Spec.Title] = rl.ID
	}

	count := 0
	for _, t := range targets {
		candID, ok, err := resolvePreSeedTarget(ctx, repos, t, roleByTitle)
		if err != nil {
			return preSeedResult{}, err
		}
		if !ok {
			continue
		}
		view, err := runner.Run(ctx, candID, 0)
		if err != nil {
			return preSeedResult{}, fmt.Errorf("seed: pre-seed agent for %s: %w", t.candidateEmail, err)
		}
		count += view.ApplicationsSubmitted
	}
	return preSeedResult{ApplicationCount: count}, nil
}

// resolvePreSeedTarget looks up the candidate by email and confirms the target
// role exists. Missing candidates or roles are skipped gracefully so the same
// runner works across seed datasets.
func resolvePreSeedTarget(
	ctx context.Context, repos Repositories, t preSeedTarget, roleByTitle map[string]kernel.ID,
) (kernel.ID, bool, error) {
	email, err := identity.NewEmail(t.candidateEmail)
	if err != nil {
		return "", false, fmt.Errorf("seed: invalid email %q: %w", t.candidateEmail, err)
	}
	user, err := repos.Users.ByEmail(ctx, email)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindNotFound {
			return "", false, nil
		}
		return "", false, err
	}
	if _, ok := roleByTitle[t.roleTitle]; !ok {
		return "", false, nil
	}
	return user.ID, true, nil
}
