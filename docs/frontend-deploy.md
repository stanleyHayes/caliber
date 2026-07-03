<!--
  CAL-152 — Frontend deploy (Vercel) + preview envs.

  This is deployment documentation for the Vite SPA in `web/`. The companion
  config is `web/vercel.json`. Neither can be applied or verified in this
  sandbox: the exact set of supported keys in `web/vercel.json` and the Git /
  preview behavior described here MUST be validated against Vercel's current
  Project Configuration schema and Git-integration docs before relying on them
  (https://vercel.com/docs/projects/project-configuration and
  https://vercel.com/docs/deployments/overview). Treat concrete values
  (build command, output dir, header names) as a starting point to confirm,
  not as verified-live settings.
-->

# Frontend deploy (Vercel) — CAL-152

How the Project Caliber SPA (`web/`, React + Vite) is built, previewed per PR,
and promoted to production on Vercel. Backend hosting (Render) and the
environment/secret model are documented elsewhere and only cross-referenced here:

- Environment topology (dev/staging/prod, what differs, no shared secrets):
  [docs/environments.md](environments.md).
- Where secrets live and how they rotate: [docs/runbooks/secret-rotation.md](runbooks/secret-rotation.md).
- Env-var templates: [.env.example](../.env.example) (dev),
  [.env.staging.example](../.env.staging.example), [.env.production.example](../.env.production.example).

## Project & build settings

Vercel builds only the `web/` subdirectory. Set the Vercel project's **Root
Directory** to `web` (Project Settings → General), so the config below and
`package.json` resolve correctly. The build is driven by `web/vercel.json`:

| Setting | Value | Source |
|---|---|---|
| Framework preset | `vite` | `vercel.json` `framework` |
| Build command | `npm run build` | `vercel.json` `buildCommand` / `web/package.json` |
| Output directory | `dist` | `vercel.json` `outputDirectory` / `vite.config.ts` `build.outDir` |
| Install command | (Vercel default: `npm install`) | derived from `package-lock.json` |

`npm run build` runs `tsc --noEmit && vite build` and then the prerender
pipeline (CAL-121): the public pages `/`, `/login`, `/register`, `/404` are
emitted as real static HTML files under `dist/`, while the authenticated app
shell stays client-rendered.

## SPA routing (rewrites)

`vercel.json` rewrites everything **except** already-built static assets to
`/index.html`, so client-side routes deep-link correctly on refresh:

```
"source": "/((?!assets/|v1/).*)", "destination": "/index.html"
```

Vercel serves an existing file from `dist/` before applying a rewrite, so the
prerendered `index.html`, `login`, `register`, and `404` pages, hashed
`/assets/*` bundles, `robots.txt`, and `sitemap.xml` are served as-is; only
unknown paths fall through to the SPA shell. `/v1/*` is excluded so an API path
is never accidentally swallowed by the SPA — see the API-origin note below.

## Security & cache headers

`vercel.json` sets long-lived immutable caching for hashed `/assets/*`,
`must-revalidate` for HTML/JSON/txt/xml, and a baseline of host-independent
security headers on every response: `X-Content-Type-Options: nosniff`,
`X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`,
a restrictive `Permissions-Policy`, and HSTS. These mirror the browser-security
posture established in CAL-114 (see [docs/threat-model.md](threat-model.md)).

**Content-Security-Policy is intentionally not set here.** A correct CSP must
name the exact API origin (`VITE_API_URL`) and any analytics host, both of
which vary per environment and are not known to a static header rule. The API
gateway already emits CSP for its own responses (CAL-114). If a page-level CSP
is added on Vercel later, template the API/analytics origins per environment and
verify it does not break the API `fetch` (`web/src/api/client.ts`) or the
optional Plausible script (CAL-128) before rolling it out.

## Environment variables (per environment)

Only `VITE_`-prefixed vars are exposed to the browser bundle; set them in Vercel
under **Project Settings → Environment Variables**, scoped to the matching Vercel
environment. Values are baked in at build time, so a change requires a redeploy.

| Var | Production | Preview (staging) | Development | Notes |
|---|---|---|---|---|
| `VITE_API_URL` | prod API origin (e.g. `https://api.projectcaliber.app`) | staging API origin (e.g. `https://api.staging.projectcaliber.app`) | empty (Vite dev proxy forwards `/v1` same-origin) | Backend base URL; consumed by `web/src/api/client.ts` / `interview.ts`. |
| `VITE_PLAUSIBLE_DOMAIN` | prod data-domain (optional) | staging domain (optional) | unset | Analytics off when blank (CAL-128). |
| `VITE_PLAUSIBLE_SCRIPT_URL` | optional override | optional override | unset | Defaults to plausible.io. |
| `VITE_PLAUSIBLE_API_URL` | optional override | optional override | unset | Defaults to plausible.io. |
| `VITE_WEB_VITALS_ENDPOINT` | optional | optional | unset | Web Vitals beacon target. |
| `VITE_SEARCH_CONSOLE_VERIFICATION` | prod token (optional) | optional | unset | Meta-tag verification. |
| `VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN` | prod token (optional) | optional | unset | HTML-file verification (build-time). |

The canonical list and descriptions live in [.env.example](../.env.example) and
`web/src/vite-env.d.ts` — this table only maps them onto Vercel's
Production / Preview / Development scopes. Cross-origin API calls require the
backend's `CALIBER_CORS_ORIGINS` to include the Vercel origin for that
environment (see [docs/environments.md](environments.md)); the preview→staging
mapping assumes preview builds point at the staging API.

## Previews on every PR (the AC)

Vercel's Git integration is enabled by connecting the repository to the Vercel
project — no extra config is required for this behavior:

- **Every push to a PR branch** produces an isolated **Preview Deployment** with
  a unique URL, using the **Preview**-scoped environment variables above. Vercel
  comments the preview URL on the PR. This satisfies the AC (previews on every PR).
- Preview builds are the natural place to point the SPA at the **staging** API
  (`VITE_API_URL` = staging origin) so a PR can be exercised end-to-end against
  staging before merge, consistent with the staging role in
  [docs/environments.md](environments.md).

## Production promotion

- **Pushes/merges to `main`** trigger a **Production Deployment** built with the
  **Production**-scoped variables, which Vercel then promotes to the production
  domain. This is Vercel's default for the configured Production Branch (`main`);
  confirm the Production Branch is set to `main` in Project Settings → Git.
- Rollback: promote a previous production deployment ("Instant Rollback") from
  the Vercel dashboard; no rebuild needed.
- This is the frontend counterpart to the backend deploy flow (CD to staging
  CAL-147, gated prod promotion CAL-148) — the two deploy independently, joined
  only by the per-environment `VITE_API_URL` / `CALIBER_CORS_ORIGINS` pairing.

## Notes / caveats

- `web/vercel.json` must stay valid JSON (no comments); all commentary lives in
  this file.
- The offline Docker/nginx path ([deploy/Dockerfile.web](../deploy/Dockerfile.web),
  [deploy/nginx.web.conf](../deploy/nginx.web.conf)) is a separate, self-hosted way
  to serve the same `dist/` output for the compose stack; Vercel is the hosted
  path. Keep SPA-fallback and header intent consistent between the two.
- Before applying: reconcile `web/vercel.json` keys and the Git/preview behavior
  with Vercel's current docs (see the note at the top of this file).
