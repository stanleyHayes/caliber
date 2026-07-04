package wiring_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
	pgrepo "github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	matchingdom "github.com/xcreativs/caliber/internal/domain/matching"
	roledom "github.com/xcreativs/caliber/internal/domain/role"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/config"
	"github.com/xcreativs/caliber/internal/platform/migrate"
	"github.com/xcreativs/caliber/internal/platform/wiring"
)

func reencryptMigrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "migrations")
}

// TestReencryptRotatesFieldKey proves CAL-117 key rotation end-to-end: PII sealed
// with an OLD key is re-sealed with a NEW key by the reencrypt pass, the plaintext
// is preserved and readable under the new key, and the retired old key can no
// longer read the rotated rows — across all four PII-bearing entity types.
func TestReencryptRotatesFieldKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "pgvector/pgvector:pg17",
		tcpostgres.WithDatabase("caliber"),
		tcpostgres.WithUsername("caliber"),
		tcpostgres.WithPassword("caliber"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("docker unavailable, skipping: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	sqlDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, migrate.Up(sqlDB, reencryptMigrationsDir(t)))
	require.NoError(t, sqlDB.Close())
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	oldKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	newKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	oldCipher, err := fieldcrypto.NewFieldCipher(oldKey)
	require.NoError(t, err)
	newCipher, err := fieldcrypto.NewFieldCipher(newKey)
	require.NoError(t, err)

	// Seed PII sealed with the OLD key: user, employer, role, candidate, profile,
	// interview (with a completed report), and a match.
	const (
		location  = "Kumasi, Ghana"
		summary   = "Ama led the payments platform team"
		evidence  = "led the payments platform team at Kuvora"
		answer    = "I built a payments API in Go"
		rationale = "Strong Go payments background; led the Kuvora team"
	)
	email, err := identity.NewEmail("rotate@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Ama", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))
	emp := kernel.NewID()
	_, err = pool.Exec(ctx, `INSERT INTO employers (id, company_name) VALUES ($1, $2)`, emp.String(), "Acme")
	require.NoError(t, err)
	rl, err := roledom.NewRole(emp,
		roledom.RoleSpec{Title: "Engineer", Seniority: roledom.SeniorityMid},
		roledom.Rubric{Competencies: []roledom.Competency{{Name: "Core", Weight: 1, MustHave: true}}},
		time.Unix(1000, 0).UTC())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewRoleRepo(pool).Create(ctx, rl))

	oldCands := pgrepo.NewCandidateRepo(pool, pgrepo.WithCandidateCipher(oldCipher))
	cand, err := talent.NewCandidate(u.ID, location, talent.CandidateIntake{Location: location, SalaryFloor: 90000, DealBreakers: []string{"no-relocation"}})
	require.NoError(t, err)
	cand.ID = u.ID
	require.NoError(t, oldCands.Create(ctx, cand))

	oldProfs := pgrepo.NewTalentProfileRepo(pool, pgrepo.WithFieldCipher(oldCipher))
	prof, err := talent.NewTalentProfile(cand.ID, summary,
		[]talent.ProfileCompetency{{Name: "backend", Level: 4, EvidenceQuote: evidence, SourceSpan: "exp#1"}})
	require.NoError(t, err)
	require.NoError(t, oldProfs.Create(ctx, prof))

	oldIvs := pgrepo.NewInterviewRepo(pool, pgrepo.WithInterviewCipher(oldCipher))
	iv, err := interviewdom.NewInterview(rl.ID, cand.ID, interviewdom.ModeText)
	require.NoError(t, err)
	require.NoError(t, iv.Transition(interviewdom.StateAsking))
	require.NoError(t, iv.Ask("What did you build?", "backend"))
	require.NoError(t, iv.Answer(answer))
	require.NoError(t, iv.Transition(interviewdom.StateScoring))
	require.NoError(t, iv.Complete(interviewdom.ReportCard{
		InterviewID: iv.ID, RoleID: rl.ID, CandidateID: cand.ID,
		Verdict: interviewdom.VerdictAdvance, Confidence: kernel.ConfidenceHigh,
		Scores:              []interviewdom.CompetencyScore{{Competency: "backend", Score: 4, Evidence: answer}},
		RecommendedNextStep: "onsite",
	}))
	require.NoError(t, oldIvs.Create(ctx, iv))

	oldMatches := pgrepo.NewMatchRepo(pool, pgrepo.WithMatchCipher(oldCipher))
	m := &matchingdom.Match{
		ID: kernel.NewID(), RoleID: rl.ID, CandidateID: cand.ID, OverallScore: 0.9, Confidence: kernel.ConfidenceHigh,
		Breakdown: []matchingdom.MatchBreakdownItem{{Competency: "backend", Score: 4, Evidence: evidence}},
		Rationale: rationale, WatchOuts: []string{"limited frontend"},
	}
	require.NoError(t, oldMatches.Upsert(ctx, m))

	// The NEW key cannot read the old-key rows yet.
	newCands := pgrepo.NewCandidateRepo(pool, pgrepo.WithCandidateCipher(newCipher))
	_, err = newCands.ByID(ctx, cand.ID)
	require.Error(t, err, "new key cannot decrypt old-key data before rotation")

	// Rotate: primary = new key, previous = old key.
	cfg := config.Config{
		DatabaseURL:                dsn,
		FieldEncryptionKey:         newKey,
		FieldEncryptionKeyPrevious: []string{oldKey},
	}
	res, err := wiring.Reencrypt(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.Equal(t, 1, res.Candidates)
	assert.Equal(t, 1, res.Profiles)
	assert.Equal(t, 1, res.Interviews)
	assert.Equal(t, 1, res.Matches)

	// The NEW key now reads every entity, plaintext preserved.
	gotCand, err := newCands.ByID(ctx, cand.ID)
	require.NoError(t, err)
	assert.Equal(t, location, gotCand.Location)
	assert.InDelta(t, 90000, gotCand.Intake.SalaryFloor, 0.001)

	gotProf, err := pgrepo.NewTalentProfileRepo(pool, pgrepo.WithFieldCipher(newCipher)).ByCandidateID(ctx, cand.ID)
	require.NoError(t, err)
	assert.Equal(t, summary, gotProf.Summary)
	require.Len(t, gotProf.Competencies, 1)
	assert.Equal(t, evidence, gotProf.Competencies[0].EvidenceQuote)

	gotIv, err := pgrepo.NewInterviewRepo(pool, pgrepo.WithInterviewCipher(newCipher)).ByID(ctx, iv.ID)
	require.NoError(t, err)
	require.Len(t, gotIv.Turns, 1)
	assert.Equal(t, answer, gotIv.Turns[0].Answer)
	require.NotNil(t, gotIv.Report)

	gotMatch, err := pgrepo.NewMatchRepo(pool, pgrepo.WithMatchCipher(newCipher)).ByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, rationale, gotMatch.Rationale)

	// The retired OLD key can no longer read the rotated rows.
	_, err = pgrepo.NewCandidateRepo(pool, pgrepo.WithCandidateCipher(oldCipher)).ByID(ctx, cand.ID)
	require.Error(t, err, "the retired key cannot read rows re-sealed with the new key")

	// Re-running is idempotent: the rotating cipher reads its own new-key writes.
	res2, err := wiring.Reencrypt(ctx, cfg, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.Equal(t, 1, res2.Candidates)
	regot, err := newCands.ByID(ctx, cand.ID)
	require.NoError(t, err)
	assert.Equal(t, location, regot.Location)
}
