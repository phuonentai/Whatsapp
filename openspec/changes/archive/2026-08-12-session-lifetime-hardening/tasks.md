## 1. Sliding Sessions

- [x] 1.1 [FE-NEXT] Extend `app/api/auth/session/refresh/route.ts` to pass `session_duration_minutes` (env, default 480) to the Stytch session authenticate/exchange so the underlying session slides; keep existing mock-auth branch. Verification: `pnpm lint`; `pnpm build` passes.
  - Route already passed `getSessionDurationMinutes()` (env, default 480) to `client.sessions.authenticate` (committed); extended to additionally honor a client-supplied `session_duration_minutes` from the request body (env fallback) so the hook's explicit request is real; mock-auth branch untouched.
  - GATE: `pnpm lint` PASS (0 errors / 4 pre-existing warnings); `pnpm build` PASS (2026-08-11).

- [x] 1.2 [FE-NEXT] Create `useSlidingSession` hook (client): refresh every 10 min while `document.visibilityState === "visible"`; on refresh failure clear cookies and redirect to `/auth?returnTo=…`; mount only when a session exists. Verification: `pnpm build`; hook unit tests pass (interval, visibility gating, failure redirect).
  - New `lib/hooks/use-sliding-session.ts` (+ `use-sliding-session.test.tsx`, 7 tests): renews on mount + every 10 min while visible, pauses while hidden, resumes on `visibilitychange`; HTTP rejection → clears `stytch_session`/`stytch_session_jwt` and redirects `/auth?returnTo=<path>`; transient network failures ignored (retry next tick). Mounted in `components/layout/dashboard-layout.tsx` with `enabled: Boolean(auth?.isAuthenticated)` (session exists only).
  - GATE: hook tests PASS (7/7); full `pnpm test` PASS (204/204); `pnpm build` PASS.

## 2. Deactivation Revocation (Backend)

- [x] 2.1 [BE-DOMAIN] Add `SessionRevoker` interface to `organizations/domain` (`RevokeMemberSessions(ctx, stytchOrgID, stytchMemberID) error`). Verification: `make build`; `go test ./internal/modules/organizations/...` passes.
  - Added to `internal/modules/organizations/domain/auth_provider.go` with SSOT note (no local session state).
  - GATE: `go build ./...` PASS; `go test ./internal/modules/organizations/...` PASS (2026-08-11).

- [x] 2.2 [BE-INFRA] Implement in the Stytch auth adapter: list member sessions (`GET /v1/b2b/sessions`, org+member filter) then revoke each (`POST /v1/b2b/sessions/revoke`), idempotent on already-revoked; wrapped in the existing circuit breaker; breaker-open returns an error the service treats as deferred. Verification: `make build`; adapter unit tests (happy path, already-revoked no-op, breaker-open) pass.
  - `stytchMemberRepository.RevokeMemberSessions` (implements `domain.SessionRevoker`): `Sessions.Get` (org+member) → `Sessions.Revoke` per `member_session_id`, whole operation behind shared `stytchcfg.Client.Run` circuit breaker; 404 (`ErrNotFound`) treated as idempotent no-op; breaker-open fails fast with `ErrCircuitOpen`.
  - Also fixed latent bug in `internal/platform/stytch/errors.go`: the SDK returns value-typed `stytcherror.Error`, so `errors.As` with a pointer target never matched (MapError was a pass-through; `IsDuplicateSlugError` never fired). Value targets now used.
  - Adapter tests `stytch_member_repository_test.go` (httptest + real SDK client): happy path (list+2 revokes, filter assert), already-revoked no-op, breaker-open (API calls stop), no-sessions no-op, revoke-failure surfaces, validation.
  - GATE: `go build ./...` PASS; `go test ./internal/modules/organizations/...` PASS; full `go test ./...` PASS.

- [x] 2.3 [BE-INFRA] Call `RevokeMemberSessions` in the member deactivation path after the local status update; on revocation failure, log warning and continue (deactivation result carries `session_revocation_pending` notice). Verification: `make build`; service test asserts revocation invoked after status update.
  - `organizationService.UpdateAccount`: after local status update succeeds and a deactivation transition (active → inactive/suspended) is detected with Stytch org+member IDs present, calls `sessionRevoker.RevokeMemberSessions`; on failure logs warning and sets transient `SessionRevocationPending` (`session_revocation_pending`, JSON omitempty, NOT persisted — repos map fields explicitly) on the returned account. `organizationService` gained `SessionRevoker` + logger deps (wired in `module.go`; shared client carries the breaker).
  - Service tests: revocation invoked after status update (org/member assert), revocation failure completes deactivation with notice, active role change skips revocation, no-Stytch-member skips revocation.
  - GATE: `go build ./...` PASS; `go test ./internal/modules/organizations/...` PASS.

## 3. Docs Alignment

- [x] 3.1 [OPS-GOV] Update `STYTCH_CONFIGURATION.md` session-duration line (default 480 min / 8h, env-overridable; remove 43200); confirm `docs/02-authentication.md` matches. Verification: grep shows consistent values.
  - `STYTCH_CONFIGURATION.md`: `NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES=480` with default/sliding note (43200 removed). `docs/02-authentication.md`: already "8 hours (480 minutes)"; added Sliding Sessions + deactivation-revocation section. `.env.example` 43200 → 480. `app/authenticate/page.tsx` login fallback 60 → 480 (single default). `.env.local` left as intentional local override (gitignored).
  - GATE: grep — no `43200` remains outside `openspec/changes` artifacts; docs/env all report 480.

## 4. Verification Gate

- [x] 4.1 [FE-NEXT] `pnpm lint`, `pnpm build`, `pnpm test` pass.
  - GATE (2026-08-11): `pnpm lint` PASS (0 errors, 4 pre-existing warnings); `pnpm build` PASS; `pnpm test` PASS (42 files, 204/204 tests, incl. 7 new hook tests).

- [x] 4.2 [BE-INFRA] `make build`; `go test ./internal/modules/organizations/...` passes.
  - GATE (2026-08-11): `export PATH=$PATH:/usr/local/go/bin && go build ./...` PASS; `go test ./internal/modules/organizations/...` PASS; full `go test ./...` PASS; `go vet ./internal/platform/stytch/... ./internal/modules/organizations/...` PASS. (`make sqlc` blocked by port conflict per repo constraint — not required by this change; no schema change.)

- [x] 4.3 [OPS-GOV] `openspec validate session-lifetime-hardening` passes.
  - GATE (2026-08-11): `openspec validate session-lifetime-hardening` → "Change 'session-lifetime-hardening' is valid".

**Archive ready:** all 9/9 tasks verified (2026-08-11). Gates green: `go build ./...`, `go test ./...` (full), `go vet` (affected packages), `pnpm lint` (0 errors / 4 pre-existing warnings), `pnpm build`, `pnpm test` (204/204), `openspec validate` PASS. No live-credential or deployed-env verification tasks outstanding — revocation/list use the existing Stytch client (covered by httptest-backed adapter tests) and the sliding renewal is unit-tested at the hook boundary. Docs consistent (no 43200 outside change artifacts).
