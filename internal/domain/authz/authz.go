// Package authz is the role-based access-control model (CAL-154): a central,
// granular permission matrix mapping each role to the capabilities it may
// exercise. It replaces ad-hoc "is this role allowed?" checks scattered across
// handlers with one explicit, testable, inspectable source of truth, and makes
// adding a role or a capability a one-line matrix edit rather than a hunt
// through the codebase. Pure domain — it depends only on identity.Role.
package authz

import (
	"slices"

	"github.com/xcreativs/caliber/internal/domain/identity"
)

// Permission is a granular capability a role may hold. Permissions are named for
// the business action they gate, not the transport, so the same permission can
// back a gRPC handler, a future admin UI, or a CLI.
type Permission string

// The capability set. Reviewer capabilities (hiring work) are held by employers
// and recruiters; candidate capabilities are self-scoped to the acting user
// (ownership is still enforced per-request — a permission grants the ability to
// act on *your own* resources, tenancy/IDOR checks do the rest, CAL-153).
const (
	PermManageRoles    Permission = "roles:manage"       // create/edit role specs + rubrics
	PermViewShortlist  Permission = "shortlist:view"     // generate/refine/view a role's shortlist
	PermRecordDecision Permission = "decision:record"    // record a hiring decision (decline)
	PermResolveContest Permission = "contest:resolve"    // resolve a candidate's assessment contest
	PermViewDashboard  Permission = "dashboard:view"     // the Talent Radar god-view
	PermReadAuditLog   Permission = "audit:read"         // read the hiring-decision audit trail
	PermScreenSelf     Permission = "interview:screen"   // start/answer one's own screening interview
	PermRunAgent       Permission = "agent:run"          // run the autonomous candidate agent
	PermManageProfile  Permission = "profile:manage"     // create/update one's own talent profile
	PermRaiseContest   Permission = "contest:raise"      // dispute an assessment about oneself
)

// matrix maps a role to the permissions it holds. It is the single source of
// truth for RBAC; adding a role (e.g. an admin) or a capability is an edit here.
//
//nolint:gochecknoglobals // a read-only RBAC lookup table — the single source of truth for authz.
var matrix = map[identity.Role][]Permission{
	identity.RoleEmployer: {
		PermManageRoles, PermViewShortlist, PermRecordDecision,
		PermResolveContest, PermViewDashboard, PermReadAuditLog,
	},
	identity.RoleRecruiter: {
		PermManageRoles, PermViewShortlist, PermRecordDecision,
		PermResolveContest, PermViewDashboard, PermReadAuditLog,
	},
	identity.RoleCandidate: {
		PermScreenSelf, PermRunAgent, PermManageProfile, PermRaiseContest,
	},
}

// Can reports whether the role holds the permission.
func Can(role identity.Role, p Permission) bool {
	return slices.Contains(matrix[role], p)
}

// PermissionsFor returns the sorted permissions a role holds — the inspectable
// view an admin tool renders, and the basis for a permissions claim.
func PermissionsFor(role identity.Role) []Permission {
	perms := slices.Clone(matrix[role])
	slices.Sort(perms)
	return perms
}

// Roles returns the roles known to the matrix, sorted, for admin tooling.
func Roles() []identity.Role {
	out := make([]identity.Role, 0, len(matrix))
	for r := range matrix {
		out = append(out, r)
	}
	slices.Sort(out)
	return out
}
