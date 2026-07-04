package authz_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xcreativs/caliber/internal/domain/authz"
	"github.com/xcreativs/caliber/internal/domain/identity"
)

func TestReviewersHoldHiringCapabilities(t *testing.T) {
	for _, r := range []identity.Role{identity.RoleEmployer, identity.RoleRecruiter} {
		assert.True(t, authz.Can(r, authz.PermManageRoles))
		assert.True(t, authz.Can(r, authz.PermViewShortlist))
		assert.True(t, authz.Can(r, authz.PermResolveContest))
		assert.True(t, authz.Can(r, authz.PermViewDashboard))
		assert.True(t, authz.Can(r, authz.PermReadAuditLog))
		assert.True(t, authz.Can(r, authz.PermViewProfile))
		assert.True(t, authz.Can(r, authz.PermViewReportCard))
		// A reviewer cannot do candidate-only things.
		assert.False(t, authz.Can(r, authz.PermScreenSelf))
		assert.False(t, authz.Can(r, authz.PermRunAgent))
		assert.False(t, authz.Can(r, authz.PermManagePrivacy))
	}
}

func TestCandidateHoldsSelfCapabilitiesOnly(t *testing.T) {
	c := identity.RoleCandidate
	assert.True(t, authz.Can(c, authz.PermScreenSelf))
	assert.True(t, authz.Can(c, authz.PermViewReportCard))
	assert.True(t, authz.Can(c, authz.PermRunAgent))
	assert.True(t, authz.Can(c, authz.PermViewProfile))
	assert.True(t, authz.Can(c, authz.PermManageProfile))
	assert.True(t, authz.Can(c, authz.PermManagePrivacy))
	assert.True(t, authz.Can(c, authz.PermRaiseContest))
	// The core RBAC guarantee: a candidate holds no reviewer capability.
	assert.False(t, authz.Can(c, authz.PermViewDashboard))
	assert.False(t, authz.Can(c, authz.PermResolveContest))
	assert.False(t, authz.Can(c, authz.PermReadAuditLog))
	assert.False(t, authz.Can(c, authz.PermManageRoles))
	assert.False(t, authz.Can(c, authz.PermViewShortlist))
}

func TestUnknownRoleHoldsNothing(t *testing.T) {
	assert.False(t, authz.Can(identity.RoleUnspecified, authz.PermViewDashboard))
	assert.Empty(t, authz.PermissionsFor(identity.RoleUnspecified))
}

func TestPermissionsForIsSorted(t *testing.T) {
	perms := authz.PermissionsFor(identity.RoleCandidate)
	assert.Len(t, perms, 7)
	for i := 1; i < len(perms); i++ {
		assert.Less(t, string(perms[i-1]), string(perms[i]), "permissions are sorted for stable admin display")
	}
}

func TestRolesEnumeratesTheMatrix(t *testing.T) {
	assert.Len(t, authz.Roles(), 4) // employer, recruiter, candidate, admin
}

func TestAdminHoldsReviewerCapabilitiesPlusUserAdmin(t *testing.T) {
	a := identity.RoleAdmin
	// Admin holds every reviewer capability...
	for _, p := range []authz.Permission{
		authz.PermManageRoles, authz.PermViewShortlist, authz.PermRecordDecision,
		authz.PermResolveContest, authz.PermViewDashboard, authz.PermReadAuditLog,
		authz.PermViewReportCard, authz.PermViewProfile,
	} {
		assert.True(t, authz.Can(a, p), "admin should hold %s", p)
	}
	// ...plus user administration, which no other role has.
	assert.True(t, authz.Can(a, authz.PermManageUsers))
	assert.False(t, authz.Can(identity.RoleEmployer, authz.PermManageUsers))
	assert.False(t, authz.Can(identity.RoleCandidate, authz.PermManageUsers))
	// Admin is an operator, not a candidate: no self-scoped candidate capabilities.
	assert.False(t, authz.Can(a, authz.PermScreenSelf))
	assert.False(t, authz.Can(a, authz.PermRunAgent))
	assert.False(t, authz.Can(a, authz.PermManageProfile))
	// The matrix now enumerates four roles.
	assert.Len(t, authz.Roles(), 4)
}
