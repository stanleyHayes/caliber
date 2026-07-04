# Enterprise SSO readiness (CAL-155)

The POC authenticates with email + password (Argon2id) and issues its own JWT
session. Enterprise customers need to bring their own identity provider (OIDC or
SAML). This document describes the **pluggability seam** that makes SSO a drop-in
behind the existing auth port — no changes to callers, tokens, or authorization.

## The seam

Authentication has two halves: **proving who the user is**, and **issuing a
Caliber session**. Only the first half differs for SSO. The `app.SSOAuthenticator`
port isolates it:

```go
// internal/app/auth.go
type SSOIdentity struct { Subject, Email, Name string }

type SSOAuthenticator interface {
    Authenticate(ctx context.Context, providerToken string) (SSOIdentity, error)
}
```

`identity.Service.LoginWithSSO(ctx, providerToken)` uses it:

1. `Authenticate` verifies the IdP assertion (an OIDC `id_token` or a SAML
   response) and returns the asserted identity, or `kernel.Unauthorized`.
2. the service maps the asserted **email** to a Caliber account
   (`UserRepository.ByEmail`),
3. and, if the account exists and is active, issues a session through the **same
   `TokenService`** as a password login.

So the rest of the platform — interceptors, `Principal`, RBAC, refresh/rotation —
is identical regardless of how the user logged in. SSO is enabled purely by
wiring an authenticator:

```go
identity.NewService(users, hasher, tokens, refresh, clock,
    identity.WithSSO(oidcAuthenticator))
```

Absent `WithSSO`, `LoginWithSSO` returns "SSO is not configured" — the seam is
inert, so this ships safely in the POC.

## Implementing a provider (post-POC)

Write an adapter under `internal/adapters/outbound/auth` implementing
`SSOAuthenticator`:

- **OIDC** — validate the `id_token` (signature against the IdP JWKS, `iss`,
  `aud`, `exp`, `nonce`), then return `SSOIdentity{Subject: sub, Email: email}`.
  A library such as `github.com/coreos/go-oidc` handles discovery + verification.
- **SAML** — validate the signed SAML response/assertion against the IdP
  metadata certificate, then map the NameID/attributes to `SSOIdentity`.

Add an inbound endpoint (a gRPC `LoginWithSSO` RPC or an OAuth redirect/callback
handler) that hands the provider token to `Service.LoginWithSSO`. Configuration
(issuer URL, client id/secret, JWKS/metadata) belongs in the secret store, read
via `config` — never in the repo (see [secret-rotation.md](runbooks/secret-rotation.md)).

## Decisions & scope

- **Account linking:** the POC maps by verified email and does **no just-in-time
  provisioning** — an SSO identity with no local account is rejected. JIT
  provisioning (auto-create on first SSO login, honoring the employer/candidate
  role) is a deliberate follow-on; `SSOIdentity.Subject` is captured so a future
  version can pin the link to the stable IdP subject rather than a mutable email.
- **Authorization is unchanged:** SSO only establishes authentication. Roles and
  per-resource ownership continue to flow through the existing RBAC + tenant
  model (CAL-116 / CAL-153).
- **No live IdP ships in the POC** (voice-style post-win scope); the seam +
  `LoginWithSSO` + tests prove pluggability without external credentials.
