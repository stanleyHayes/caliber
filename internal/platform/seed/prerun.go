package seed

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xcreativs/caliber/internal/app"
	interviewapp "github.com/xcreativs/caliber/internal/app/interview"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
)

// PreRunResult summarises how many screening interviews were completed during
// demo seeding.
type PreRunResult struct {
	InterviewCount int
}

// preRunTarget identifies a candidate/role pair to pre-screen.
type preRunTarget struct {
	candidateEmail string
	roleTitle      string
}

// generatedPreRunTargets returns the first two hero pairs so Flow A shortlists
// already carry real assessments, while leaving at least one hero (Esi) to run
// live in Flow B.
func generatedPreRunTargets() []preRunTarget {
	return []preRunTarget{
		{candidateEmail: "ama.mensah.hero@example.com", roleTitle: "Senior Backend Engineer"},
		{candidateEmail: "kofi.asante.hero@example.com", roleTitle: "Data Engineer"},
	}
}

// handCuratedPreRunTargets mirrors the hero-pair idea for the hand-curated demo
// dataset. Ama and Kofi are pre-run; Esi and Yaw remain live.
func handCuratedPreRunTargets() []preRunTarget {
	return []preRunTarget{
		{candidateEmail: "ama.mensah@example.com", roleTitle: "Senior Backend Engineer"},
		{candidateEmail: "kofi.asante@example.com", roleTitle: "Data Engineer"},
	}
}

// preRunInterviews runs screening interviews to completion for the supplied
// targets, storing report cards so the demo shortlists show real assessments.
// Targets whose candidate email is not present are skipped gracefully, so the
// same runner works across seed datasets.
func preRunInterviews(
	ctx context.Context, repos Repositories, llm app.LLMClient, targets []preRunTarget,
) (PreRunResult, error) {
	if llm == nil {
		return PreRunResult{}, nil
	}
	if repos.Interviews == nil {
		return PreRunResult{}, errors.New("seed: interview repository is required for pre-run interviews")
	}

	interviewer := interviewapp.NewInterviewer(
		repos.Roles, repos.Interviews, llm,
		interviewdom.Config{MaxQuestions: 2},
		interviewapp.WithPassportUpdater(repos.Profiles),
	)

	roles, _, err := repos.Roles.ListOpen(ctx, kernel.NewPage(1, 1000))
	if err != nil {
		return PreRunResult{}, fmt.Errorf("seed: list roles for pre-run: %w", err)
	}
	roleByTitle := make(map[string]kernel.ID, len(roles))
	for _, rl := range roles {
		roleByTitle[rl.Spec.Title] = rl.ID
	}

	count := 0
	for _, t := range targets {
		candID, roleID, ok, err := resolvePreRunTarget(ctx, repos, t, roleByTitle)
		if err != nil {
			return PreRunResult{}, err
		}
		if !ok {
			continue
		}
		if err := runInterviewToReport(ctx, repos, interviewer, candID, roleID); err != nil {
			return PreRunResult{}, fmt.Errorf("seed: pre-run interview for %s: %w", t.candidateEmail, err)
		}
		count++
	}
	return PreRunResult{InterviewCount: count}, nil
}

func resolvePreRunTarget(
	ctx context.Context, repos Repositories, t preRunTarget, roleByTitle map[string]kernel.ID,
) (kernel.ID, kernel.ID, bool, error) {
	email, err := identity.NewEmail(t.candidateEmail)
	if err != nil {
		return "", "", false, fmt.Errorf("seed: invalid email %q: %w", t.candidateEmail, err)
	}
	user, err := repos.Users.ByEmail(ctx, email)
	if err != nil {
		if kernel.KindOf(err) == kernel.KindNotFound {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	roleID, ok := roleByTitle[t.roleTitle]
	if !ok {
		return "", "", false, nil
	}
	return user.ID, roleID, true, nil
}

func runInterviewToReport(
	ctx context.Context,
	repos Repositories,
	interviewer *interviewapp.Interviewer,
	candidateID, roleID kernel.ID,
) error {
	profile, err := repos.Profiles.ByCandidateID(ctx, candidateID)
	if err != nil {
		return err
	}

	iv, q, err := interviewer.Start(ctx, roleID, candidateID, interviewdom.ModeText)
	if err != nil {
		return err
	}

	for q != nil {
		answer := answerForCompetency(q.CompetencyTag, profile)
		var report *interviewdom.ReportCard
		q, report, err = interviewer.Answer(ctx, iv.ID, answer)
		if err != nil {
			return err
		}
		if report != nil {
			return nil
		}
	}
	return errors.New("seed: interview ended without a report card")
}

func answerForCompetency(competency string, profile *talent.TalentProfile) string {
	for _, c := range profile.Competencies {
		if strings.EqualFold(c.Name, competency) && c.EvidenceQuote != "" {
			return fmt.Sprintf("I have applied %s in production; %s.", c.Name, c.EvidenceQuote)
		}
	}
	return fmt.Sprintf("I have practical experience with %s from multiple production projects.", competency)
}
