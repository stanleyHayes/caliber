# Multi-tenancy & tenant isolation (CAL-153)

Caliber is a POC that several employers use at once. This document states the
tenancy model, where isolation is enforced, how it is tested, and the known
POC limitations.

## Model: the employer-user *is* the tenant

The POC does not introduce a separate `Organization`/`Tenant` entity. An
**employer is a user** (`identity.RoleEmployer` / `identity.RoleRecruiter`), and
every employer-owned resource carries that user's id as its owner:

- `role.Role.EmployerID` is the owning employer user's id (the seed sets it so,
  and registration has no separate employer entity).
- A `matching.Match` and an interview `ReportCard` both carry a `RoleID`, so
  their owner is resolved through the role: `match/report → role → EmployerID`.

The **isolation invariant** is therefore a single, uniform rule:

> A caller may act on an employer-owned resource only when
> `resource.EmployerID == principal.UserID`.

The acting principal is always read from the authenticated context
(`app.Principal`), never trusted from the request body. This keeps the model
free of a JWT/claims change: no tenant id is needed in the token because the
user id already is the tenant boundary.

## Where isolation is enforced

Enforcement lives in the **use-case layer** (app), so it is adapter-agnostic and
unit-tested, with the gRPC handlers passing `principal.UserID` in from the auth
context. Candidate-owned resources are scoped candidate-self by the same
principle (`candidate.ID == principal.UserID` for registered candidates).

| Resource / action | Rule | Enforced in | Story |
|---|---|---|---|
| GenerateRoleSpec / UpdateRoleSpec / ListRoles | `role.EmployerID == caller` | role handlers + use-cases | CAL-116 |
| GenerateShortlist / RefineShortlist / RecordRejection | `role.EmployerID == caller` (before any recall/scoring) | matching use-cases | CAL-116 |
| GetReportCard (reviewer branch) | reviewer owns the screened role (`EmployerForInterview`) | interview handler | CAL-116 |
| **ResolveContest** | reviewer owns the role behind the contested match/report card | **contest use-case (`Service.Resolve`)** | **CAL-153** |
| RunAgent / TimeAdvance / GetWakeUpView / ListApplications | candidate-self | agent handler | CAL-116 |
| CreateProfileFromCV / GetTalentProfile | candidate-self / self-or-owning-reviewer | talent handler | CAL-116 |
| StartInterview / SubmitAnswer / GetReportCard (candidate) | candidate owns the interview | interview handler | CAL-116 |

### ResolveContest (this change)

A contest disputes an assessment — a shortlist `Match` or an interview
`ReportCard`. `Service.Resolve` now resolves the contest's subject to its owning
employer before mutating anything:

1. load the contest,
2. `SubjectMatch` → `MatchRepository.ByID(subjectID)` → `match.RoleID`;
   `SubjectReportCard` → `InterviewRepository.ByID(subjectID)` → `interview.RoleID`,
3. `RoleRepository.ByID(roleID)` → `role.EmployerID`,
4. reject with `kernel.Forbidden` unless `EmployerID == reviewerID` — **before**
   the state transition or any audit write.

This also validates the subject: a `subject_id` that references no real
assessment surfaces `NotFound` instead of silently resolving. `MatchRepository`
gained a `ByID` method (Postgres + in-memory adapters, sqlc query, mock) to make
this lookup possible — it was the missing data-model piece CAL-116 flagged.

## How it is tested

Cross-tenant isolation is proven at both the use-case and handler layers:

- `internal/app/contest` — a reviewer from another employer resolving a contest
  is rejected with `Forbidden` before any `Update`/audit; the owner succeeds; the
  report-card subject resolves through the interview; a missing subject surfaces
  `NotFound`.
- `internal/adapters/inbound/grpc` — a non-owning employer gets
  `codes.PermissionDenied` on `ResolveContest`; the owning employer succeeds.
- CAL-116 already carries cross-employer IDOR tests for role, shortlist,
  rejection, and report-card reads.

## Known POC limitations (honest)

- **`ListAuditLog` cross-tenant reads.** The audit row records only
  `entity` + `entity_id` (e.g. a candidate or contest id), not the owning role,
  so per-tenant scoping is not derivable from audit data alone — and naive
  actor-scoping breaks the legitimate contest trail (a reviewer must see the
  candidate's *raise* plus their own *resolve*). It stays **reviewer-only (RBAC)
  and append-only** as the compensating control. Closing it needs an owner/tenant
  column on audit rows (enrich-going-forward) or the same subject→role resolution
  applied per entity type — tracked as a follow-up.
- **Recruiter ↔ employer membership.** There is no junction table linking a
  recruiter to an employer, so each employer/recruiter *user* is its own tenant
  (`role.EmployerID == principal.UserID`). A shared-organization model where
  several recruiters act for one employer needs a membership table and is out of
  POC scope.
- **App-level scoping, not Postgres RLS.** Isolation is enforced by ownership
  checks in the use-cases (consistently, and unit-tested) rather than database
  row-level security. RLS is a defense-in-depth hardening for a later stage.
