package wiring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
)

// reencryptPageSize bounds each batch read during a re-encryption pass. It must
// not exceed kernel.MaxPageSize, or NewPage would clamp the page smaller than
// requested — the termination check counts DELIVERED rows to be robust to that,
// but keeping the request within the cap avoids wasted round-trips.
const reencryptPageSize = kernel.MaxPageSize

// ReencryptResult reports how many rows of each PII-bearing entity were rewritten.
type ReencryptResult struct {
	Candidates int
	Profiles   int
	Interviews int
	Matches    int
}

// Reencrypt rewrites every PII-bearing row through the configured field cipher,
// re-sealing it with the current primary key (CAL-117 key rotation / enablement).
// Reads use the primary key plus any CALIBER_FIELD_ENCRYPTION_KEY_PREVIOUS
// fallbacks, so rows sealed by the retiring key decrypt; writes re-seal with the
// new primary key. It is idempotent (safe to re-run) and never rewrites a row it
// could not fully decrypt — a decrypt failure aborts with that row identified,
// rather than corrupting or downgrading data. See docs/runbooks/key-rotation.md.
func Reencrypt(ctx context.Context, cfg config.Config, log *slog.Logger) (ReencryptResult, error) {
	var res ReencryptResult
	if cfg.DatabaseURL == "" {
		return res, errors.New("reencrypt: CALIBER_DATABASE_URL is required")
	}
	// Refuse to run without a primary key: a passthrough cipher cannot re-seal, so
	// running it here is never the intended operation and risks leaving PII in a
	// mixed/unencrypted state. Rotation always has a primary key configured.
	if strings.TrimSpace(cfg.FieldEncryptionKey) == "" {
		return res, errors.New("reencrypt: refusing to run without CALIBER_FIELD_ENCRYPTION_KEY")
	}

	repos, cleanup, _, err := openRepositories(ctx, cfg, log)
	if err != nil {
		return res, err
	}
	defer cleanup()
	if repos.Pool == nil {
		return res, errors.New("reencrypt: postgres persistence is required")
	}

	if err := reencryptCandidates(ctx, repos, &res); err != nil {
		return res, err
	}
	if err := eachPage(ctx, reencryptPageSize, repos.Profiles.List, func(p *talent.TalentProfile) error {
		res.Profiles++
		return repos.Profiles.Update(ctx, p)
	}); err != nil {
		return res, fmt.Errorf("reencrypt: talent profiles: %w", err)
	}

	log.Info("reencrypt complete",
		"candidates", res.Candidates, "profiles", res.Profiles,
		"interviews", res.Interviews, "matches", res.Matches)
	return res, nil
}

// reencryptCandidates rewrites every candidate and, per candidate, their
// interviews and matches (all PII is reachable from the candidate root).
func reencryptCandidates(ctx context.Context, repos Repositories, res *ReencryptResult) error {
	return eachPage(ctx, reencryptPageSize, repos.Candidates.List, func(c *talent.Candidate) error {
		if err := repos.Candidates.Update(ctx, c); err != nil {
			return fmt.Errorf("reencrypt: candidate %s: %w", c.ID, err)
		}
		res.Candidates++

		if err := eachPage(ctx, reencryptPageSize,
			func(ctx context.Context, page kernel.Page) ([]*interviewdom.Interview, int64, error) {
				return repos.Interviews.ByCandidate(ctx, c.ID, page)
			},
			func(iv *interviewdom.Interview) error {
				res.Interviews++
				return repos.Interviews.Update(ctx, iv)
			}); err != nil {
			return fmt.Errorf("reencrypt: interviews for candidate %s: %w", c.ID, err)
		}

		return eachPage(ctx, reencryptPageSize,
			func(ctx context.Context, page kernel.Page) ([]*matchingdom.Match, int64, error) {
				return repos.Matches.ForCandidate(ctx, c.ID, page)
			},
			func(m *matchingdom.Match) error {
				res.Matches++
				return repos.Matches.Upsert(ctx, m)
			})
	})
}

// eachPage invokes fn for every item across all pages of a paginated lister. It
// terminates on the number of rows actually DELIVERED reaching the reported total
// (or an empty page), NOT on an assumed page size: the page layer clamps the size
// to kernel.MaxPageSize, so counting page*pageSize would overshoot and silently
// skip the tail of a table larger than one page. Rewriting rows in place preserves
// the row count and ordering, so this does not loop.
func eachPage[T any](
	ctx context.Context, pageSize int,
	list func(context.Context, kernel.Page) ([]T, int64, error),
	fn func(T) error,
) error {
	processed := int64(0)
	for page := 1; ; page++ {
		items, total, err := list(ctx, kernel.NewPage(page, pageSize))
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := fn(item); err != nil {
				return err
			}
			processed++
		}
		if len(items) == 0 || processed >= total {
			return nil
		}
	}
}
