# Role-based access control (CAL-154)

Authorization is a single, granular **permission matrix**
(`internal/domain/authz`) rather than role checks scattered across handlers.
Handlers declare the *capability* they need; the matrix decides which roles hold
it. Adding a role or a capability is a one-line edit in one place.

## Model

- **Permission** — a granular capability named for the business action it gates
  (`roles:manage`, `shortlist:view`, `contest:resolve`, `dashboard:view`,
  `audit:read`, `interview:screen`, `agent:run`, `profile:manage`,
  `contest:raise`). Named for the action, not the transport, so the same
  permission backs a gRPC handler, a future admin UI, or a CLI.
- **matrix** — `role → []Permission`. The single source of truth:
  - **employer / recruiter** (reviewers): manage roles, view/refine shortlists,
    record hiring decisions, resolve contests, view the Talent Radar, read the
    audit trail.
  - **candidate**: screen (their own interview), run their agent, manage their
    own profile, raise contests about themselves.
- **`Can(role, permission) bool`** — the check. **`PermissionsFor(role)`** and
  **`Roles()`** expose the matrix for admin tooling / inspection.

Permissions grant the *ability* to act; per-resource **ownership** (a
candidate acts only on their own data; an employer only on roles they own) is
enforced separately by the tenant model (CAL-116 / CAL-153). RBAC answers "may
this role do X at all?"; tenancy answers "on *this* resource?".

## Enforcement

`grpcadapter.RequirePermission(ctx, perm)` authenticates the caller and checks
the matrix — the permission-based counterpart to `RequireRole`. The Talent Radar
dashboard is gated this way (`requireReviewer` → `RequirePermission(...,
PermViewDashboard)`); the same test suite (candidate → PermissionDenied, anon →
Unauthenticated) still passes, now driven by the matrix.

The remaining handlers keep their equivalent `RequireRole` guards; migrating each
to `RequirePermission` is mechanical and behavior-preserving (the matrix encodes
the same allow-lists), done incrementally to keep each change small and reviewed.

## Adding a role or capability

- **New capability:** add a `Permission` constant, grant it to the appropriate
  roles in `matrix`, and call `RequirePermission` (or `Can`) at the guard point.
- **New role (e.g. `admin`):** add the value to the `identity.Role` enum (and its
  `UserRole` proto peer — an *additive* enum value, backward-compatible) and a
  `matrix` row. An admin would hold a superset (e.g. all reviewer permissions
  plus `users:manage`). This is the one part that touches the locked proto, so it
  is a deliberate additive follow-on rather than folded into this change.

## Admin tooling

`PermissionsFor(role)` / `Roles()` render the effective matrix — the basis for an
admin screen or a `caliber roles` CLI that shows "who can do what". A management
UI for editing role↔permission assignments at runtime (vs. the compile-time
matrix) is a post-POC follow-on; the POC matrix is code-reviewed and versioned,
which is the stronger default for a small, fixed role set.
