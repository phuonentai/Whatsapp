# Design: fix-spec-validation-debt

## Context

`openspec validate --specs` currently reports 38 of 54 living specs with issues. Investigation established the real state (all claims verified against the repo):

- **2 ERROR-level specs**:
  - `frontend-api-client/spec.md` — empty file (0 lines). The capability exists in code: `next_b2b_starter/lib/api/api/client/api-client.ts` (733 lines) + 14 typed repositories under `next_b2b_starter/lib/api/api/repositories/`.
  - `lean-data-isolation/spec.md` — requirement "Optional PostgreSQL RLS policy for defense-in-depth" violates the requirement-text rule (statement must contain SHALL or MUST). The statement is "The system MAY add PostgreSQL Row-Level Security policies…".
- **33 WARNING-level specs** — `## Purpose` under 50 characters (stub lines like "Specification for tag management."). Validator minimum is 50 chars.
- **2 INFO-level specs** — over-long requirement text in `stytch-authorization` and `vertical-playbooks`.
- **Drift** — `stytch-authorization` living spec lost 4 requirements present in the archived `2026-07-28-stytch-debt-cleanup` delta (Role normalization, RBAC API endpoint authentication, RBACService implementation backed by Stytch policy, DTOs retained as API contract) and implemented in code (`stytch_rbac_service.go`, `rbac_policy.go`, RBAC routes).
- **CI gate exists but is ineffective** — `.github/workflows/ci.yml` `spec-validation` job runs `npx openspec validate --specs`, which exits 0 despite failures. `--strict` exits 1 on brief-Purpose warnings but **misses structural errors** (verified: empty spec validates `True` under `--strict`). Neither mode alone catches everything.

## Goals / Non-Goals

**Goals:**
- Make `openspec validate --specs` report 54/54 passing with zero issues at any severity level.
- Restore the 4 lost `stytch-authorization` requirements without resurrecting the abandoned password-auth feature.
- Give `frontend-api-client` a real spec authored from the verified implementation.
- Make the CI gate actually fail on regressions (both structural errors and framing warnings).
- Preserve all requirement content in the 33 Purpose-expanded specs — framing edits only.

**Non-Goals:**
- No application code, DB schema, API surface, or Stytch tenant policy changes.
- No re-architecture of the OpenSpec fold/archive process.
- No content rewrite beyond the one validator-error reword and the verified restore.
- Not adopting the stale `normalize-spec-format` change (its `crm-core-data` drift claim is false — those requirements already exist in the living spec at lines 157/177/197).

## Decisions

### D1. Purpose expansion rule
Each of the 33 stub Purposes is replaced with a single descriptive sentence >50 chars that states the capability's behavioural contract, sourced from the spec's own requirements (never invented). Format: one paragraph under `## Purpose`, no heading changes elsewhere. Requirement bodies untouched. Verified count: 33 specs have Purpose 1–49 chars and a WARNING; 17 specs pass with no issues; 2 are INFO-only.

### D2. frontend-api-client spec source of truth
Authored from `next_b2b_starter/lib/api/api/client/api-client.ts` and repository layer. Contract elements (all verified in code):
- `ApiClient` class with `get/post/put/patch/delete`, base URL resolution: server-side `API_BASE_URL_INTERNAL` (default `http://localhost:8080/api`), client-side `NEXT_PUBLIC_API_BASE_URL` (default `/api`).
- Bearer token attachment via `resolveAccessToken()` unless `skipAuth` or explicit `Authorization` header; `credentials: "include"`.
- Mock-auth E2E forwarding of `X-Test-Org-ID` when `AUTH_MOCK_ENABLED === "true"` (server: next/headers cookies; client: document cookie).
- 401 handling: classify attached token state, `resolveAccessToken({forceRefresh:true})` or `refreshToken()`, single retry, `logoutUser()` on failure, redirect to login with `returnTo`.
- Token resolution: in-memory cache → stored cookie (`SESSION_JWT_COOKIE_NAME`) → refresh; refresh with retry (max 3, backoff 1s/2s/4s, shared promise 10s timeout); server refresh exchanges Stytch session token via `sessions.authenticate` (480 min); browser refresh hits `POST /api/auth/session/refresh`.
- JSON envelope `{ data?: T; success?: boolean }` unwrapped by repository-local `unwrap()`.
- Repositories pattern: `const BASE = "/path"` + exported repository object of typed calls.

### D3. stytch-authorization restore
Delta uses ADDED Requirements with the 4 requirements + scenarios taken verbatim from the archived `2026-07-28-stytch-debt-cleanup` delta (already merged history). Password-auth requirement deliberately excluded (abandoned by `2026-08-08-abandon-password-auth`).

### D4. CI gate hardening
`spec-validation` job runs both modes:
```yaml
- name: Run spec validation
  run: |
    npx openspec validate --specs
    npx openspec validate --specs --strict
```
Non-strict catches structural errors; strict catches brief-Purpose warnings. Any non-zero exit fails the job. (Verified: non-strict exits 0 on errors; strict exits 1 on warnings — the two are complementary.)

### D5. INFO trims
`stytch-authorization` and `vertical-playbooks`: where a requirement body exceeds 500 chars purely from editorial padding, split long sentences without changing meaning. Framing-only; no requirement text semantics change. If a trim risks changing meaning, leave the text and accept the INFO (target is zero, but semantics beat cosmetics).

### D6. Change-owned delta specs
Only 4 delta spec files, matching content-changing capabilities: `frontend-api-client` (new), `lean-data-isolation` (modified requirement), `stytch-authorization` (added requirements), `spec-validation` (modified CI-gate requirement). The 33 Purpose expansions and 2 INFO trims are framing-only and are NOT delta specs (per governance, deltas exist for requirement behaviour, not framing).

## Risks / Trade-offs

- **Parallel active changes** (`add-whatsapp-embedded-signup`, `add-mercadopago-billing`, `add-siigo-invoicing`) may edit some of the 33 specs mid-flight. Their deltas touch requirement content, not `## Purpose` framing; conflict risk low, resolve by keeping both changes' framing + requirements.
- **Purpose length drift**: >50 chars is a hard validator floor; reviewers may cut Purposes back under it. The hardened CI gate is the guard.
- **frontend-api-client spec staleness**: authored from today's code; future client changes must update the spec (CI gate only checks format, not drift — drift detection remains a manual review concern, same as every other capability).
- **Strict-mode blind spot** resolved by dual-run (D4); running strict alone would miss empty/malformed specs.
- **INFO trims** could touch requirement wording; mitigated by D5's semantics-first rule.
