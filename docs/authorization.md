# Authorization model (CAL-116)

How Project Caliber decides **who may do what** to **which resource**. This is the
implemented model (complements the requirements in [threat-model.md](threat-model.md)
§E and the data-protection posture in [data-protection.md](data-protection.md)).

Three checks compose, in this order, at the inbound (gRPC) boundary:

1. **Authentication** — a valid bearer access token injects an `app.Principal`
   (`UserID`, `Role`) into the context. Enforced by the unary interceptor
   (`NewAuthInterceptor`) and — because unary interceptors do **not** run for
   streaming RPCs — the streaming interceptor (`NewAuthStreamInterceptor`), which
   wraps the `ServerStream` so the handler reads the principal from
   `stream.Context()`.
2. **Authorization (RBAC)** — `RequireRole(ctx, …)` enforces the caller holds an
   allowed role (employer / recruiter / candidate). Unauthenticated → `Unauthorized`;
   authenticated-but-wrong-role → `Forbidden`.
3. **Ownership (anti-IDOR)** — the caller may act only on **their own** resources.
   The acting identity is always `principal.UserID` from the token, **never** an id
   taken from the request body.

> **Golden rule:** a body id (`employer_id`, `candidate_id`, `role_id`) is only ever
> a *target to compare against* the token identity — it is never *trusted as* the
> actor. Every ownership check runs **before** any state read-back or mutation.

## The identity model (why ownership is a simple equality)

The POC has **no separate tenant/employer entity**. Two invariants make ownership a
direct id comparison — no tenant table, no JWT change:

- **Employers are users.** A role's `EmployerID` is the owning user's id (the seed
  sets it so; registration creates the role under the registrant's id). So
  *"does this employer own this role?"* is `principal.UserID == role.EmployerID`.
- **A candidate is their user.** The provisioner sets `candidate.ID == user.ID`. So
  *"is this the owning candidate?"* is `principal.UserID == candidateID`.

Helpers in `auth_interceptor.go`: `requireSelfEmployer`, `requireSelfCandidate`,
`requireSelfCandidateOrReviewer` (reviewer = employer/recruiter, used only for the
pool-wide Talent Radar profile read).

## Per-service authorization matrix

| RPC | AuthN | Role | Ownership |
|---|---|---|---|
| Identity.Register / Login / Refresh | public | — | — |
| Identity.GetMe / Logout | required | any | self (token) |
| Role.GenerateRoleSpec / ListRoles | required | employer/recruiter | `employer_id == UserID` |
| Role.UpdateRoleSpec | required | employer/recruiter | loads role, `EmployerID == UserID` |
| Role.GetRole | required | any | — (candidates view postings to apply) |
| Matching.GenerateShortlist / RefineShortlist | required | employer/recruiter | loads role, `EmployerID == actorUserID` (before recall/scoring/mutation) |
| Matching.RecordRejection | required | employer/recruiter | loads role, `EmployerID == actorUserID` (before the audit write) |
| Interview.StartInterview (stream) | required | candidate | self (`candidate_id == UserID`) |
| Interview.SubmitAnswer | required | candidate | owns the interview (`CandidateForInterview == UserID`) |
| Interview.GetReportCard | required | candidate **or** owning employer | candidate→`card.CandidateID==UserID`; reviewer→`EmployerForInterview==UserID` |
| Talent.CreateProfileFromCV | required | candidate | self |
| Talent.GetTalentProfile | required | candidate **or** reviewer | self, or any reviewer (Talent Radar pool view) |
| CandidateAgent.RunAgent / TimeAdvance / GetWakeUpView / ListApplications | required | candidate | self |
| Dashboard.* (pool / supply-demand / alerts / time-to-shortlist) | required | employer/recruiter | — (pool-wide reviewer view) |
| Contest.RaiseContest / ListMyContests | required | candidate | self |
| Contest.ResolveContest | required | employer/recruiter | **none yet** — see Deferred |
| Audit.ListAuditLog | required | employer/recruiter | **none yet** — see Deferred |

Each row is covered by an IDOR/authz test (cross-actor → `PermissionDenied`,
anonymous → `Unauthenticated`).

## Report cards are private (the subtle one)

`GetReportCard` must not use the shared `requireSelfCandidateOrReviewer` helper: that
helper grants **any** reviewer, which would leak a candidate's Flow B verdict, scores,
and evidence to employers who never posted the role nor ran the screening. Instead the
handler scopes the reviewer branch to the role owner via `Interviewer.EmployerForInterview`
(interview → `RoleID` → `role.EmployerID`). The talent-profile read keeps the
any-reviewer helper on purpose — the Talent Radar pool is meant to be reviewer-wide.

## Deferred (tracked toward CAL-153)

Two **read/resolve** paths are intentionally left RBAC-only because correct ownership
is **unresolvable from the current data model**, not because the check was forgotten:

- **Contest.ResolveContest** — a contest stores `subject` + `subject_id` only; mapping
  that to the owning role needs `MatchRepository.ByID` (does not exist; matches are
  keyed by `(role, candidate)`) or a report-card store. Note the contest flow has **no
  frontend** — the endpoint is unreachable in the demo.
- **Audit.ListAuditLog** — audit rows carry only `entity` + `entity_id` (a candidate /
  contest / application id), not the owning role. Actor-scoping was tried and reverted:
  it breaks the legitimate contest trail (a reviewer must see the candidate's *raise*
  plus their own *resolve*). Correct per-entity role-ownership needs the same lookups as
  ResolveContest.

**Compensating controls:** both remain reviewer-only (RBAC), the audit trail is
append-only, and every privileged action is itself audited.

## Verification

- Per-RPC IDOR/authz unit tests across all nine services.
- End-to-end authn acceptance test through the real Argon2id hasher + JWT service
  (`TestAuthFlowEndToEnd`).
- Two adversarial multi-agent review passes (find → independent-skeptic verify →
  synthesize) over the whole authorization surface; the second confirmed every write
  path + candidate-self path airtight and surfaced the report-card leak (now fixed).
