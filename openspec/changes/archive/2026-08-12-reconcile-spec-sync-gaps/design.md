# Design: reconcile-spec-sync-gaps

## Context

A gap analysis (2026-08-12) compared all archived change deltas against the living `openspec/specs/` tree and found behavioral requirements that were implemented and tested but never folded into the authoritative specs:

1. `add-e2e-edge-coverage` (archived 2026-08-10 in bulk commit `b04d685`) — 6 requirements describing E2E coverage. Premise-verified: the tests exist (`surrounding-processes.spec.ts` covers cross-org isolation / mock-auth guard / RBAC boundary / pagination / outbound reply persistence; `whatsapp-edge-cases.spec.ts` covers webhook edge cases; `inbox-ui.spec.ts` covers pagination UI).
2. `fix-frontend-build` (archived 2026-08-08) — 2 requirements describing `ApiClient` query-param serialization and `queryKeys.whatsappConfig`. Premise-verified: `lib/api/api/client/api-client.ts` implements `params`/`URLSearchParams`/`buildUrl`; `lib/hooks/queries/query-keys.ts` exports the `whatsappConfig` namespace with `all`/`detail()`; hooks `use-whatsapp-config-query.ts`, `use-toggle-whatsapp-config.ts`, `use-upsert-whatsapp-config.ts` consume them.
3. `auth-email-check` — the `GET /api/auth/check-email` endpoint is live (`internal/modules/organizations/member_handler.go`, route registered in `routes.go`) and consumed by `app/auth/page.tsx:142`, but no living capability documents its contract. The capability spec of the same name was archived in `fix-auth-cross-origin-csp` and never re-established.

Constraints: OpenSpec living specs are the behavioural source of truth (AGENTS.md). Folded requirements must match reality — the drift being fixed is spec-silence, not code. No Stytch API contract or tenant policy changes; the endpoint is a local DB read.

## Goals / Non-Goals

**Goals**
- Living specs record all three behaviors above (8 requirements + 1 new capability).
- Auth page resolves its API base URL through the shared ApiClient configuration (same-origin default), removing the hardcoded cross-origin fallback that re-introduced the `fix-auth-cross-origin-csp` regression.
- All spec changes validate under `openspec validate --specs` and preserve existing requirements.

**Non-Goals**
- No local credential storage or access (per constitution + proposal).
- No change to check-email 200/404/400/500 semantics.
- No anti-enumeration enforcement (distinct 404 message stays as-is; product decision deferred — see Open Questions).
- No folding of abandoned `password-auth` deltas or one-time `mvp-launch` governance deltas.
- No re-verification of the E2E suite itself.

## Decisions

### D1. Fold mechanism: agent-driven merge into living specs (not `openspec archive`)

The source changes are already archived; `openspec archive` is not applicable. The fold follows `openspec-sync-specs` semantics: read the delta, apply requirements to the living spec, preserve unrelated content.

- **Why**: this repo's archive history shows programmatic folds can drop structure (documented `dabc95f` `## Requirements` header-drop defect) and are destructive to requirements added by *other* changes if the spec file is regenerated. Agent-driven merge is idempotent and surgical.
- **Alternatives considered**: `openspec archive --skip-specs` + CLI fold — rejected (defect history); manual wholesale spec rewrite — rejected (risk of losing unrelated requirements).

### D2. Requirement content: delta text as source, premise-verified

For `crm-test-infrastructure`, `crm-whatsapp-e2e`, and `frontend-api-client`, the delta requirements are copied into the living specs with at most naming/consistency edits:

- `crm-test-infrastructure` (5) + `crm-whatsapp-e2e` (1): the `add-e2e-edge-coverage` delta text is used as-is; each requirement's premise (an E2E test exercising the behavior) was verified against the current Playwright suite.
- `frontend-api-client` (2): delta text used as-is; the referenced hook files (`use-whatsapp-config-query.ts`, `use-toggle-whatsapp-config.ts`, `use-upsert-whatsapp-config.ts`) and `queryKeys.whatsappConfig` all exist today under `lib/hooks/`.

- **Why**: the deltas are the archived contract; the code and tests confirm they describe reality.
- **Alternatives considered**: rewriting the requirements to current spec style (e.g., splitting "is E2E-tested" into per-capability assertions) — rejected for scope; the E2E-tested requirement form is the established pattern in these capabilities.

### D3. `auth-email-check` capability: document current contract, minimal frontend fix

New capability `auth-email-check` with two requirements:

1. **Endpoint contract** (as implemented): `GET /api/auth/check-email?email=<addr>` → 200 when an active account exists (org-agnostic local DB lookup via `localOrgRepo.GetByUserEmail`), 404 when not found, 400 when `email` param is missing, 500 on repository failure. No session required; no Stytch API call; no credential material in the response.
2. **Same-origin base URL**: the auth page SHALL resolve the API base URL via the shared ApiClient configuration (client-side default relative `/api`) instead of a hardcoded cross-origin fallback.

Frontend fix in `app/auth/page.tsx`: replace
```ts
const apiBaseUrl = (process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080/api").replace(/\/$/, "");
```
with `apiClient.getBaseUrl()` (client-side resolution, same-origin default per `ApiClient` constructor), keeping the existing raw `fetch` + explicit `404`/`!ok` branching unchanged.

- **Why `apiClient.getBaseUrl()` over `apiClient.get(...)`**: `ApiClient.request` throws on non-2xx and `buildApiError` embeds the status only in the error message (no structured `status` field), so preserving the explicit 404 branch would require fragile message parsing. `getBaseUrl()` (already exported, line ~85) yields the same resolved base URL with a one-line change.
- **Why not keep the hardcoded fallback**: it defaults to cross-origin `http://localhost:8080/api` in production browsers, contradicting `frontend-api-client`'s same-origin rule and the intent of the archived `fix-auth-cross-origin-csp`.
- **Alternatives considered**: (a) default to relative `/api` inline — rejected: diverges from ApiClient's server/client resolution; (b) `apiClient.get` + parse `API Error 404` — rejected: fragile.

### D4. Verification: spec validation + affected-code gates

- Specs: `openspec validate --specs` (strict) — the CI spec-validation gate.
- Backend: no code change; `go build ./...` as a sanity gate only.
- Frontend: `pnpm lint`, `npx tsc --noEmit`, `pnpm build`, and the auth-page unit test (`app/auth/page.test.tsx` if present, else a targeted vitest run of the auth flow) — the base-URL change must not alter 200/404/400 behavior.
- E2E: not run (requires full stack); the covered behaviors already exist in the suite and are unchanged.

## Risks / Trade-offs

- [Enumeration tension] The spec will document the current distinct-404 behavior; `stytch-nextjs-components` mandates a neutral anti-enumeration message → Mitigation: recorded as an Open Question; the `auth-email-check` requirement text does not prescribe the frontend message copy, only the status-code contract, so the two specs do not contradict.
- [Spec drift if auth page changes again] The base-URL requirement depends on the shared client config; a future refactor could re-introduce a hardcoded URL → Mitigation: the requirement is explicit ("SHALL resolve via the shared ApiClient configuration"), and the frontend-api-client same-origin rule independently covers it.
- [Fold errors] Appending to living specs could clobber requirements added by other changes (e.g., `crm-test-infrastructure` has 9 requirements from several changes) → Mitigation: agent-driven merge only touches the named requirement blocks; verify with `openspec validate --specs` and a requirements-count diff before/after.
- [check-email endpoint behavior change] The handler/service are untouched; only the spec records behavior → no runtime risk.

## Migration Plan

1. Apply the 5+1 E2E coverage requirements to `crm-test-infrastructure` / `crm-whatsapp-e2e`.
2. Apply the 2 requirements to `frontend-api-client`.
3. Create `openspec/specs/auth-email-check/spec.md` (new capability).
4. Apply the one-line base-URL change to `app/auth/page.tsx`.
5. Run verification gates (D4).
6. Commit as one change; on failure, `git revert` the spec + page changes (specs are additive; no data migration).

Rollback: Git revert of the commit (specs + page). No Stytch tenant state involved.

## Open Questions

1. **Anti-enumeration**: `app/auth/page.tsx` currently shows a distinct error on 404 (enumeration signal), contradicting the neutral-message mandate in `stytch-nextjs-components` ("Login page renders a custom email form"). Should a follow-up change enforce anti-enumeration (neutral message + no distinct 404 branch)? Deferred — out of scope here, but flagged for council/product.
2. **`check-email` auth posture**: the endpoint is currently anonymous (no session). Should it remain so, or require a session for rate-limit/abuse control? Current rate limiting for magic-link sends exists (`sign-in-rate-limiting`); the check endpoint itself is unthrottled. Recorded, not changed here.
