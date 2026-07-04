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
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestTalentProfileSummaryEncryptedAtRest proves CAL-117 field encryption: with a
// key configured, the profile summary is stored as ciphertext in Postgres, the
// repo transparently returns plaintext, and a repo without the key cannot read it.
func TestTalentProfileSummaryEncryptedAtRest(t *testing.T) {
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

	// A candidate to own the profile.
	email, err := identity.NewEmail("enc.cand@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Kofi", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))
	cand, err := talent.NewCandidate(u.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	cand.ID = u.ID
	require.NoError(t, pgrepo.NewCandidateRepo(pool).Create(ctx, cand))

	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	cipher, err := fieldcrypto.NewFieldCipher(key)
	require.NoError(t, err)
	repo := pgrepo.NewTalentProfileRepo(pool, pgrepo.WithFieldCipher(cipher))

	const summary = "Ama built and shipped a payments API in Go, cutting latency 40%."
	prof, err := talent.NewTalentProfile(cand.ID, summary, nil)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, prof))

	// The raw column is ciphertext, not the plaintext summary.
	var raw string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT summary FROM talent_profiles WHERE id=$1`, prof.ID.String()).Scan(&raw))
	assert.NotEqual(t, summary, raw, "the stored summary must not be plaintext")
	assert.True(t, strings.HasPrefix(raw, "enc:v1:"), "the stored summary is versioned ciphertext, got %q", raw)

	// Reading through the cipher-backed repo returns the plaintext.
	got, err := repo.ByCandidateID(ctx, cand.ID)
	require.NoError(t, err)
	assert.Equal(t, summary, got.Summary)

	// A repo without the key cannot recover the plaintext — the ciphertext is opaque.
	opaque, err := pgrepo.NewTalentProfileRepo(pool).ByCandidateID(ctx, cand.ID)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(opaque.Summary, "enc:v1:"), "without the key the summary stays opaque")
}
