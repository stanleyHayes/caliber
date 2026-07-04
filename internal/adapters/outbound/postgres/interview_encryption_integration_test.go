package postgres_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"strings"
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
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestInterviewQAEncryptedAtRest proves CAL-117 field encryption for the
// screening dialogue: with a key configured, both the answered turn and the
// pending question are stored as ciphertext in interview_turns, the repo
// transparently returns plaintext, and a repo without the key cannot read them.
func TestInterviewQAEncryptedAtRest(t *testing.T) {
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
	email, err := identity.NewEmail("iv.enc@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Ama", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))
	cand, err := talent.NewCandidate(u.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	cand.ID = u.ID
	require.NoError(t, pgrepo.NewCandidateRepo(pool).Create(ctx, cand))

	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	cipher, err := fieldcrypto.NewFieldCipher(key)
	require.NoError(t, err)
	repo := pgrepo.NewInterviewRepo(pool, pgrepo.WithInterviewCipher(cipher))

	const (
		q1     = "What did you build?"
		answer = "I built a payments API in Go."
		q2     = "How did you test it?"
	)
	iv, err := interviewdom.NewInterview(rl.ID, cand.ID, interviewdom.ModeText)
	require.NoError(t, err)
	require.NoError(t, iv.Transition(interviewdom.StateAsking))
	require.NoError(t, iv.Ask(q1, "backend"))
	require.NoError(t, repo.Create(ctx, iv))
	// Answer the first question and ask a second (pending) one.
	require.NoError(t, iv.Answer(answer))
	require.NoError(t, iv.Ask(q2, "testing"))
	require.NoError(t, repo.Update(ctx, iv))

	// Every stored question/answer is versioned ciphertext, not plaintext.
	rows, err := pool.Query(ctx,
		`SELECT question, answer FROM interview_turns WHERE interview_id=$1`, iv.ID.String())
	require.NoError(t, err)
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var rawQ string
		var rawA sql.NullString
		require.NoError(t, rows.Scan(&rawQ, &rawA))
		seen++
		assert.True(t, strings.HasPrefix(rawQ, "enc:v1:"), "question stored as ciphertext, got %q", rawQ)
		assert.NotContains(t, rawQ, "build", "plaintext question must not leak")
		if rawA.Valid {
			assert.True(t, strings.HasPrefix(rawA.String, "enc:v1:"), "answer stored as ciphertext, got %q", rawA.String)
			assert.NotContains(t, rawA.String, "payments", "plaintext answer must not leak")
		}
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, 2, seen, "one answered turn + one pending question")

	// The cipher-backed repo transparently returns plaintext.
	got, err := repo.ByID(ctx, iv.ID)
	require.NoError(t, err)
	require.Len(t, got.Turns, 1)
	assert.Equal(t, q1, got.Turns[0].Question)
	assert.Equal(t, answer, got.Turns[0].Answer)
	require.NotNil(t, got.Pending)
	assert.Equal(t, q2, got.Pending.Text)

	// A repo without the key cannot recover the plaintext Q&A while there is no
	// report card yet — the turns are TEXT fields, so they stay opaque without error.
	opaque, err := pgrepo.NewInterviewRepo(pool).ByID(ctx, iv.ID)
	require.NoError(t, err)
	require.Len(t, opaque.Turns, 1)
	assert.True(t, strings.HasPrefix(opaque.Turns[0].Question, "enc:v1:"), "without the key the question stays opaque")
	assert.True(t, strings.HasPrefix(opaque.Turns[0].Answer, "enc:v1:"), "without the key the answer stays opaque")

	// Complete the interview: the report card carries transcript-quote evidence
	// (candidate PII), which must be encrypted at rest like the turns.
	const cardEvidence = "candidate described contract-testing the payments API"
	require.NoError(t, iv.Answer("With contract tests."))
	require.NoError(t, iv.Transition(interviewdom.StateScoring))
	require.NoError(t, iv.Complete(interviewdom.ReportCard{
		InterviewID: iv.ID, RoleID: rl.ID, CandidateID: cand.ID,
		Verdict:    interviewdom.VerdictAdvance,
		Confidence: kernel.ConfidenceHigh,
		Scores: []interviewdom.CompetencyScore{
			{Competency: "testing", Score: 4, Evidence: cardEvidence},
		},
		RecommendedNextStep: "onsite loop",
	}))
	require.NoError(t, repo.Update(ctx, iv))

	// The report_card column is ciphertext; the denormalized confidence stays plain.
	var rawCard, rawConfidence string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT report_card::text, confidence FROM talent_interviews WHERE id=$1`, iv.ID.String()).
		Scan(&rawCard, &rawConfidence))
	assert.Contains(t, rawCard, "enc:v1:", "the report card is stored as wrapped ciphertext")
	assert.NotContains(t, rawCard, "contract-testing", "the report-card evidence must not leak")
	assert.Equal(t, "high", rawConfidence, "confidence is denormalized in plaintext for queryability")

	// The cipher-backed repo returns the plaintext report card.
	done, err := repo.ByID(ctx, iv.ID)
	require.NoError(t, err)
	require.NotNil(t, done.Report)
	require.Len(t, done.Report.Scores, 1)
	assert.Equal(t, cardEvidence, done.Report.Scores[0].Evidence)

	// Now that an encrypted report card exists, a keyless repo cannot load the
	// interview at all — it fails closed on the undecryptable report card.
	_, err = pgrepo.NewInterviewRepo(pool).ByID(ctx, iv.ID)
	require.Error(t, err, "a keyless repo cannot recover the encrypted report card")
}
