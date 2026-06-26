package provisioning_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/app/provisioning"
	identitydom "github.com/xcreativs/caliber/internal/domain/identity"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	"github.com/xcreativs/caliber/internal/mocks"
)

func user(t *testing.T, role identitydom.Role) *identitydom.User {
	t.Helper()
	e, err := identitydom.NewEmail("u@example.com")
	require.NoError(t, err)
	u, err := identitydom.NewUser(e, role, "U", "hash", time.Unix(1, 0))
	require.NoError(t, err)
	return u
}

func TestProvisionCandidateCreatesContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	u := user(t, identitydom.RoleCandidate)

	var created *talent.Candidate
	candidates.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *talent.Candidate) error { created = c; return nil })

	require.NoError(t, provisioning.NewCandidateProvisioner(candidates).Provision(context.Background(), u))
	require.NotNil(t, created)
	assert.Equal(t, u.ID, created.UserID, "candidate context is owned by the user")
	assert.Equal(t, u.ID, created.ID, "candidate id mirrors the user id")
}

func TestProvisionEmployerIsNoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl) // Create must never be called
	for _, role := range []identitydom.Role{identitydom.RoleEmployer, identitydom.RoleRecruiter} {
		require.NoError(t, provisioning.NewCandidateProvisioner(candidates).Provision(context.Background(), user(t, role)))
	}
}

func TestProvisionPropagatesRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	candidates := mocks.NewMockCandidateRepository(ctrl)
	candidates.EXPECT().Create(gomock.Any(), gomock.Any()).Return(kernel.Conflict("dup"))
	err := provisioning.NewCandidateProvisioner(candidates).Provision(context.Background(), user(t, identitydom.RoleCandidate))
	assert.Equal(t, kernel.KindConflict, kernel.KindOf(err))
}
