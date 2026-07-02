package postgres_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	pgrepo "github.com/xcreativs/caliber/internal/adapters/outbound/postgres"
	privacyapp "github.com/xcreativs/caliber/internal/app/privacy"
	auditdom "github.com/xcreativs/caliber/internal/domain/audit"
	"github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/platform/migrate"
)

// TestPostgresErasureCascade proves the CAL-118 right-to-erasure cascade works on
// the Postgres stack: a candidate's scoped records are deleted, the candidate row
// is removed, the owning user is anonymised in place, and the audit trail is
// retained-but-tombstoned. It wires the same privacy ports the composition root
// uses, so it also guards buildEraser's backend-agnostic assertions.
func TestPostgresErasureCascade(t *testing.T) {
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

	users := pgrepo.NewUserRepo(pool)
	candidates := pgrepo.NewCandidateRepo(pool)
	profiles := pgrepo.NewTalentProfileRepo(pool)
	apps := pgrepo.NewApplicationRepo(pool)
	matches := pgrepo.NewMatchRepo(pool)
	auditRepo := pgrepo.NewAuditRepo(pool)

	// A candidate with a user account, a talent profile, and an audit entry naming
	// them as the actor.
	email, err := identity.NewEmail("erase.me@example.com")
	require.NoError(t, err)
	u, err := identity.NewUser(email, identity.RoleCandidate, "Ama Mensah", "hash", time.Now())
	require.NoError(t, err)
	require.NoError(t, users.Create(ctx, u))

	cand, err := talent.NewCandidate(u.ID, "Accra", talent.CandidateIntake{})
	require.NoError(t, err)
	cand.ID = u.ID // registered candidates: candidate id == user id
	require.NoError(t, candidates.Create(ctx, cand))

	prof, err := talent.NewTalentProfile(cand.ID, "Backend engineer.", nil)
	require.NoError(t, err)
	require.NoError(t, profiles.Create(ctx, prof))

	entry, err := auditdom.NewAuditEntry(cand.ID, "profile.build", "candidate", cand.ID, "", "", time.Now())
	require.NoError(t, err)
	require.NoError(t, auditRepo.Append(ctx, entry))

	// Wire the cascade exactly as buildEraser does (via the privacy ports). apps
	// and matches carry no rows here, so their scoped deletes run as valid no-ops —
	// exercising every erasure query against the real schema.
	eraser := privacyapp.NewEraser(candidates, users, auditRepo, profiles, apps, matches)
	require.NoError(t, eraser.EraseCandidate(ctx, cand.ID))

	// The candidate and its scoped records are gone.
	_, err = candidates.ByID(ctx, cand.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err), "candidate row deleted")
	_, err = profiles.ByCandidateID(ctx, cand.ID)
	assert.Equal(t, kernel.KindNotFound, kernel.KindOf(err), "talent profile deleted")

	// The user row is retained but anonymised in place.
	got, err := users.ByID(ctx, u.ID)
	require.NoError(t, err, "user row retained (referenced elsewhere)")
	assert.Empty(t, got.Name, "name scrubbed")
	assert.True(t, strings.HasPrefix(got.Email.String(), "erased+"), "email tombstoned, got %q", got.Email)
	assert.Contains(t, got.Email.String(), "@erased.invalid")

	// The audit trail is retained but the subject is tombstoned.
	entries, _, err := auditRepo.List(ctx, "candidate", cand.ID, kernel.NewPage(1, 10))
	require.NoError(t, err)
	require.NotEmpty(t, entries, "audit entry retained")
	assert.Equal(t, kernel.ID("erased"), entries[0].ActorUserID, "actor tombstoned")
}
