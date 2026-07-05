# 2026-07-05 Current UI Slice Review

Scope: role navigation/detail updates, interview role selection, and the candidate agent applications panel.

## Findings

1. Medium: the applications panel could crash on unknown application statuses.
   - File: `web/src/components/agent/ApplicationsList.tsx`
   - Problem: the component indexed status metadata, labels, and colors directly with `a.status`. TypeScript only protects compile-time callers; a newer API enum value or malformed row could make `statusMeta[a.status]` undefined and crash the panel.
   - Fix: added fallback helpers so unknown statuses render as `Unknown` with the existing pending state instead of breaking the page.
   - Regression: added an Agent page test that renders an application with a runtime-only status value.

2. No blocking issues found in the reviewed role and interview paths.
   - Role delete is guarded by backend ownership checks and exposed through the same role service boundary.
   - Roles list and role detail back navigation are pinned to stable destinations, avoiding browser-history loops.
   - The interview role picker no longer shows raw permission errors when a user cannot list managed roles and still leaves the role-ID fallback available.

## Verification

- `cd web && npm run test:run -- AgentPage.test.tsx`
- `cd web && npm run lint`
- `cd web && npx tsc --noEmit`
- `git diff --check`
