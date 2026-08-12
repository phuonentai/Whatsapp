# Tasks: reconcile-spec-sync-gaps

## 1. Fold E2E coverage deltas into living specs

- [x] 1.1 [OPS-GOV] Add the 5 `add-e2e-edge-coverage` requirements to `openspec/specs/crm-test-infrastructure/spec.md`: cross-organization data isolation is E2E-tested, pagination behavior is E2E-tested, outbound reply persistence is E2E-tested, mock-auth guard is E2E-tested, RBAC boundary is E2E-tested. Preserve the 9 existing requirements unchanged. Verification: `openspec validate crm-test-infrastructure` passes; requirement count = 14; diff shows only additions.
- [x] 1.2 [OPS-GOV] Add the 1 `add-e2e-edge-coverage` requirement (webhook edge-case scenarios are E2E-tested) to `openspec/specs/crm-whatsapp-e2e/spec.md`. Preserve the 7 existing requirements. Verification: `openspec validate crm-whatsapp-e2e` passes; requirement count = 8; premise confirmed: `next_b2b_starter/e2e/specs/whatsapp-edge-cases.spec.ts` exercises the edge cases.

## 2. Fold frontend-api-client deltas into living specs

- [x] 2.1 [OPS-GOV] Add the 2 `fix-frontend-build` requirements to `openspec/specs/frontend-api-client/spec.md`: "API client supports query parameters" and "Query key namespaces match hook usage". Preserve the existing 4 requirements. Verification: `openspec validate frontend-api-client` passes; requirement count = 6; premise confirmed: `lib/api/api/client/api-client.ts` implements `params`/`URLSearchParams`, `lib/hooks/queries/query-keys.ts` exports `queryKeys.whatsappConfig` with `all`/`detail()`, and hooks `use-whatsapp-config-query.ts` / `use-toggle-whatsapp-config.ts` / `use-upsert-whatsapp-config.ts` consume them.

## 3. Create auth-email-check capability

- [x] 3.1 [OPS-GOV] Create `openspec/specs/auth-email-check/spec.md` from the delta spec with both requirements (endpoint contract + same-origin base URL), including the Purpose section. Verification: `openspec validate auth-email-check` passes; contract matches `MemberHandler.CheckEmail` (200 exists / 404 not found / 400 missing param / 500 repository failure; local DB read via `localOrgRepo.GetByUserEmail`, no Stytch API call, no session required).

## 4. Auth page base URL fix

- [x] 4.1 [FE-NEXT] In `next_b2b_starter/app/auth/page.tsx`, replace the hardcoded base-URL fallback (`process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080/api"`) with the shared client resolution `apiClient.getBaseUrl()` (import the `apiClient` singleton from `lib/api/api/client/api-client.ts`). Keep the explicit `status === 404` and `!ok` branches unchanged. Verification: `npx tsc --noEmit` passes; grep shows no `http://localhost:8080/api` literal in `app/auth/page.tsx`.

## 5. Verification gate

- [x] 5.1 [OPS-GOV] Run `openspec validate --specs` (strict). Verify: 91 specs + 1 new = 92 passed, 0 failed.
  - GATE (2026-08-12): `openspec validate --specs` → Totals: 92 passed, 0 failed (92 items).
- [x] 5.2 [BE-INFRA] Run `go build ./...` in `go-b2b-starter/`. Verify: exit 0 (no backend code changed; sanity gate only).
  - GATE (2026-08-12): `go build ./...` → exit 0.
- [x] 5.3 [FE-NEXT] Run `pnpm lint`, `npx tsc --noEmit`, and `pnpm build` in `next_b2b_starter/`. Verify: lint 0 errors (4 pre-existing baseline warnings acceptable), tsc exit 0, build exit 0.
  - GATE (2026-08-12): `pnpm lint` → 0 errors / 4 pre-existing baseline warnings; `npx tsc --noEmit` → exit 0; `pnpm build` → exit 0.
- [x] 5.4 [FE-NEXT] Run the auth-page test: `pnpm exec vitest run app/auth/page.test.tsx` (or the auth flow unit tests). Verify: passes; 200/404/400 behavior preserved.
  - GATE (2026-08-12): no `app/auth/page.test.tsx` exists; ran the auth-flow unit tests instead — `pnpm exec vitest run app/signup/page.test.tsx lib/auth/magic-link-limiter.test.ts lib/auth/audit.test.ts` → 3 files / 23 tests passed. The base-URL change preserves the explicit `status === 404` and `!ok` branches (verified by code review).

## 6. Archive decision

- [x] 6.1 [OPS-GOV] Record the archive decision in this file (`/opsx-archive` or `**Archive deferred:** <reason>`). Verify: entry present.
  - **Archive deferred:** all gates green (5.1–5.4 above); archiving performed by the orchestrator via `/opsx-archive` after this apply session.
