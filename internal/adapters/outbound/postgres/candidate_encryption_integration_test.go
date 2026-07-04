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

// TestCandidatePIIEncryptedAtRest proves CAL-117 field encryption for candidate
// PII: with a key configured, the location column and the intake preferences
// (salary floor, deal-breakers) are stored as ciphertext, the repo transparently
// returns plaintext, and a repo without the key cannot read them.
func TestCandidatePIIEncryptedAtRest(t *testing.T) {
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

	email, err := identity.NewEmail("cand.enc@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Kwame", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u))

	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{5}, 32))
	cipher, err := fieldcrypto.NewFieldCipher(key)
	require.NoError(t, err)
	repo := pgrepo.NewCandidateRepo(pool, pgrepo.WithCandidateCipher(cipher))

	const (
		location    = "Kumasi, Ghana"
		dealBreaker = "no-relocation"
	)
	intake := talent.CandidateIntake{
		TargetTitles:   []string{"Staff Engineer"},
		Location:       location,
		SalaryFloor:    120000,
		SalaryCurrency: "USD",
		DealBreakers:   []string{dealBreaker},
	}
	cand, err := talent.NewCandidate(u.ID, location, intake)
	require.NoError(t, err)
	cand.ID = u.ID
	require.NoError(t, repo.Create(ctx, cand))

	// The raw location column and preferences payload are ciphertext, not plaintext.
	var rawLoc string
	var rawPrefs []byte
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT location, preferences::text FROM candidates WHERE id=$1`, cand.ID.String()).
		Scan(&rawLoc, &rawPrefs))
	assert.True(t, strings.HasPrefix(rawLoc, "enc:v1:"), "location stored as ciphertext, got %q", rawLoc)
	assert.NotContains(t, rawLoc, "Kumasi", "plaintext location must not leak")
	prefsStr := string(rawPrefs)
	assert.NotContains(t, prefsStr, dealBreaker, "plaintext deal-breaker must not leak")
	assert.NotContains(t, prefsStr, "120000", "plaintext salary floor must not leak")
	assert.Contains(t, prefsStr, "enc:v1:", "preferences stored as wrapped ciphertext")

	// The cipher-backed repo transparently returns plaintext.
	got, err := repo.ByID(ctx, cand.ID)
	require.NoError(t, err)
	assert.Equal(t, location, got.Location)
	assert.InDelta(t, intake.SalaryFloor, got.Intake.SalaryFloor, 0.001)
	assert.Equal(t, intake.DealBreakers, got.Intake.DealBreakers)
	assert.Equal(t, intake.TargetTitles, got.Intake.TargetTitles)

	// A repo without the key cannot recover the PII: the wrapped intake ciphertext
	// passes straight through the keyless cipher and then fails to parse back into
	// the intake JSON, so the read fails closed rather than returning garbage.
	opaque, err := pgrepo.NewCandidateRepo(pool).ByID(ctx, cand.ID)
	require.Error(t, err, "a keyless repo cannot recover the wrapped intake payload")
	assert.Nil(t, opaque)

	// Isolate the location TEXT column from the intake: a second candidate with an
	// EMPTY intake still has its location encrypted at rest, so the location's
	// at-rest encryption is proven independent of the (separately-encrypted) intake
	// JSONB that carries the keyless-read failure above.
	email2, err := identity.NewEmail("loc.only@example.com")
	require.NoError(t, err)
	u2, err := identity.NewUser(email2, identity.RoleCandidate, "Efua", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, pgrepo.NewUserRepo(pool).Create(ctx, u2))
	locOnly, err := talent.NewCandidate(u2.ID, "Takoradi, Ghana", talent.CandidateIntake{})
	require.NoError(t, err)
	locOnly.ID = u2.ID
	require.NoError(t, repo.Create(ctx, locOnly))

	var rawLoc2 string
	require.NoError(t, pool.QueryRow(ctx, `SELECT location FROM candidates WHERE id=$1`, locOnly.ID.String()).Scan(&rawLoc2))
	assert.True(t, strings.HasPrefix(rawLoc2, "enc:v1:"), "location encrypted at rest independent of intake, got %q", rawLoc2)
	assert.NotContains(t, rawLoc2, "Takoradi", "plaintext location must not leak")
}
