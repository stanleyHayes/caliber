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
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/matching"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestMatchAssessmentEncryptedAtRest proves CAL-117 field encryption for the
// match assessment prose: with a key configured, the rationale, breakdown
// evidence, and watch-outs are stored as ciphertext, the repo returns plaintext,
// and a keyless repo fails closed on the encrypted JSONB.
func TestMatchAssessmentEncryptedAtRest(t *testing.T) {
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

	// Seed FK parents: employer + role, and a user-backed candidate.
	emp := kernel.NewID()
	_, err = pool.Exec(ctx, `INSERT INTO employers (id, company_name) VALUES ($1, $2)`, emp.String(), "Acme")
	require.NoError(t, err)
	rl := mkRole(t, emp, "Engineer", time.Unix(1000, 0).UTC())
	require.NoError(t, pgrepo.NewRoleRepo(pool).Create(ctx, rl))
	email, err := identity.NewEmail("match.enc@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Ama", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))
	cand, err := talent.NewCandidate(u.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	cand.ID = u.ID
	require.NoError(t, pgrepo.NewCandidateRepo(pool).Create(ctx, cand))

	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{6}, 32))
	cipher, err := fieldcrypto.NewFieldCipher(key)
	require.NoError(t, err)
	repo := pgrepo.NewMatchRepo(pool, pgrepo.WithMatchCipher(cipher))

	const (
		rationale = "Strong Go payments background; led the Kuvora platform team"
		evidence  = "shipped a payments API cutting latency 40%"
		watchOut  = "limited frontend exposure"
	)
	m := &matching.Match{
		ID: kernel.NewID(), RoleID: rl.ID, CandidateID: cand.ID,
		OverallScore: 0.87, Confidence: kernel.ConfidenceHigh,
		Breakdown: []matching.MatchBreakdownItem{
			{Competency: "backend", Score: 4, Evidence: evidence},
		},
		Rationale: rationale,
		WatchOuts: []string{watchOut},
	}
	require.NoError(t, repo.Upsert(ctx, m))

	// The rationale, breakdown, and watch_outs columns are ciphertext at rest.
	var rawRationale, rawBreakdown, rawWatchOuts string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT rationale, breakdown::text, watch_outs::text FROM matches WHERE id=$1`, m.ID.String()).
		Scan(&rawRationale, &rawBreakdown, &rawWatchOuts))
	assert.True(t, strings.HasPrefix(rawRationale, "enc:v1:"), "rationale stored as ciphertext, got %q", rawRationale)
	assert.NotContains(t, rawRationale, "Kuvora", "plaintext rationale must not leak")
	assert.Contains(t, rawBreakdown, "enc:v1:", "breakdown stored as wrapped ciphertext")
	assert.NotContains(t, rawBreakdown, "latency", "plaintext breakdown evidence must not leak")
	assert.Contains(t, rawWatchOuts, "enc:v1:", "watch_outs stored as wrapped ciphertext")
	assert.NotContains(t, rawWatchOuts, "frontend", "plaintext watch-out must not leak")

	// The cipher-backed repo transparently returns plaintext.
	got, err := repo.ByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, rationale, got.Rationale)
	require.Len(t, got.Breakdown, 1)
	assert.Equal(t, evidence, got.Breakdown[0].Evidence)
	assert.Equal(t, []string{watchOut}, got.WatchOuts)

	// A keyless repo fails closed on the encrypted breakdown/watch_outs JSONB.
	_, err = pgrepo.NewMatchRepo(pool).ByID(ctx, m.ID)
	require.Error(t, err, "a keyless repo cannot recover the encrypted match assessment")
}
