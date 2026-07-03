package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	pgrepo "github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	"github.com/xcreativs/caliber/internal/domain/identity"
	interviewdom "github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestInterviewRepoRoundTrip proves CAL-066: an interview (transcript + pending
// question + report card) round-trips through the Postgres stack across the whole
// Flow B state machine, and the right-to-erasure delete cascades to turns.
func TestInterviewRepoRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	skipIfNoDocker(t)
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
	require.NoError(t, migrate.Up(sqlDB, migrationsDir(t)))
	require.NoError(t, sqlDB.Close())

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// Seed the FK parents: an employer + role, and a user-backed candidate.
	emp := kernel.NewID()
	_, err = pool.Exec(ctx, `INSERT INTO employers (id, company_name) VALUES ($1, $2)`, emp.String(), "Acme")
	require.NoError(t, err)
	rl := mkRole(t, emp, "Engineer", time.Unix(1000, 0).UTC())
	require.NoError(t, pgrepo.NewRoleRepo(pool).Create(ctx, rl))

	email, err := identity.NewEmail("cand@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))
	cand, err := talent.NewCandidate(u.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	cand.ID = u.ID // registered candidate: candidate id == user id
	require.NoError(t, pgrepo.NewCandidateRepo(pool).Create(ctx, cand))

	repo := pgrepo.NewInterviewRepo(pool)

	// 1) Create an open interview; StartedAt must round-trip.
	iv, err := interviewdom.NewInterview(rl.ID, cand.ID, interviewdom.ModeText)
	require.NoError(t, err)
	iv.StartedAt = time.Unix(2000, 0).UTC()
	require.NoError(t, repo.Create(ctx, iv))

	got, err := repo.ByID(ctx, iv.ID)
	require.NoError(t, err)
	assert.Equal(t, interviewdom.StateOpen, got.State)
	assert.Equal(t, interviewdom.ModeText, got.Mode)
	assert.Equal(t, rl.ID, got.RoleID)
	assert.Equal(t, cand.ID, got.CandidateID)
	assert.True(t, got.StartedAt.Equal(iv.StartedAt), "StartedAt round-trips")
	assert.Empty(t, got.Turns)
	assert.Nil(t, got.Pending)
	assert.Nil(t, got.Report)

	_, err = repo.ByID(ctx, kernel.NewID())
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))

	// 2) Ask a question — the pending (unanswered) question must persist.
	require.NoError(t, iv.Transition(interviewdom.StateAsking))
	require.NoError(t, iv.Ask("What did you build?", "backend"))
	require.NoError(t, repo.Update(ctx, iv))

	got, err = repo.ByID(ctx, iv.ID)
	require.NoError(t, err)
	assert.Equal(t, interviewdom.StateAsking, got.State)
	require.NotNil(t, got.Pending, "pending question survives a reload")
	assert.Equal(t, "What did you build?", got.Pending.Text)
	assert.Equal(t, "backend", got.Pending.CompetencyTag)
	assert.Equal(t, 1, got.Pending.Ordinal)
	assert.Empty(t, got.Turns)

	// 3) Answer, then ask again: one completed turn + a new pending question.
	require.NoError(t, iv.Answer("I built a payments API."))
	require.NoError(t, iv.Ask("How did you test it?", "testing"))
	require.NoError(t, repo.Update(ctx, iv))

	got, err = repo.ByID(ctx, iv.ID)
	require.NoError(t, err)
	require.Len(t, got.Turns, 1)
	assert.Equal(t, "What did you build?", got.Turns[0].Question)
	assert.Equal(t, "I built a payments API.", got.Turns[0].Answer)
	assert.Equal(t, "backend", got.Turns[0].CompetencyTag)
	require.NotNil(t, got.Pending)
	assert.Equal(t, "How did you test it?", got.Pending.Text)
	assert.Equal(t, 2, got.Pending.Ordinal)

	// 4) Answer, score, and complete — the report card must persist.
	require.NoError(t, iv.Answer("With contract tests."))
	require.NoError(t, iv.Transition(interviewdom.StateScoring))
	card := interviewdom.ReportCard{
		InterviewID: iv.ID, RoleID: rl.ID, CandidateID: cand.ID,
		Verdict:    interviewdom.VerdictAdvance,
		Confidence: kernel.ConfidenceHigh,
		Scores: []interviewdom.CompetencyScore{
			{Competency: "backend", Score: 4, Evidence: "built a payments API"},
		},
		RecommendedNextStep: "onsite loop",
	}
	require.NoError(t, iv.Complete(card))
	require.NoError(t, repo.Update(ctx, iv))

	got, err = repo.ByID(ctx, iv.ID)
	require.NoError(t, err)
	assert.Equal(t, interviewdom.StateClosed, got.State)
	require.Len(t, got.Turns, 2)
	assert.Nil(t, got.Pending, "no pending question after completion")
	require.NotNil(t, got.Report)
	assert.Equal(t, interviewdom.VerdictAdvance, got.Report.Verdict)
	assert.Equal(t, kernel.ConfidenceHigh, got.Report.Confidence)
	require.Len(t, got.Report.Scores, 1)
	assert.Equal(t, "backend", got.Report.Scores[0].Competency)
	assert.Equal(t, "onsite loop", got.Report.RecommendedNextStep)

	var conf string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT confidence FROM talent_interviews WHERE id=$1`, iv.ID.String()).Scan(&conf))
	assert.Equal(t, "high", conf, "confidence denormalized for queryability")

	// 5) Listing returns the full aggregate; an update to a missing interview is NotFound.
	list, total, err := repo.ByCandidate(ctx, cand.ID, kernel.NewPage(1, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, iv.ID, list[0].ID)
	assert.Len(t, list[0].Turns, 2)

	ghost, err := interviewdom.NewInterview(rl.ID, cand.ID, interviewdom.ModeText)
	require.NoError(t, err)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(repo.Update(ctx, ghost)))

	// 6) Right-to-erasure: the interview and its turns are gone.
	require.NoError(t, repo.DeleteByCandidate(ctx, cand.ID))
	_, err = repo.ByID(ctx, iv.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err))
	var turnCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM interview_turns WHERE interview_id=$1`, iv.ID.String()).Scan(&turnCount))
	assert.Equal(t, 0, turnCount, "turns cascade-deleted with the interview")
}
