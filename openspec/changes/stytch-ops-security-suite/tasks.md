# Stytch Ops & Security Suite — Tasks

## 1. Stytch Project Configuration (test project)

- [ ] 1.1 [OPS-GOV] In the Stytch test project: enable impersonation; register a webhook endpoint (test URL) and copy the `whsec_` signing secret into env; configure event-log streaming destinations (Datadog site+API key and/or Grafana Loki endpoint) — record the free-beta status and which destinations were chosen. Verification: impersonation option present in dashboard; webhook endpoint appears in dashboard; streaming destination shows healthy.
- [ ] 1.2 [OPS-GOV] Verify the Svix signature scheme against the test endpoint using the Svix verification tooling with a sample `direct.member.create` event. Verification: valid signature accepted, tampered payload rejected, stale timestamp rejected (unit/httptest against the real verifier).

## 2. Backend — Webhook Ingress

- [ ] 2.1 [BE-INFRA] Implement the Svix standard-webhook verifier (HMAC-SHA256 over `{Webhook-Id}.{Webhook-Timestamp}.{body}`, `whsec_` secret from env, constant-time compare, replay window on timestamp) — reuse the existing HMAC patterns (Meta `X-Hub-Signature-256`). Unit tests: valid, tampered payload, wrong secret, stale timestamp. Verification: `make build`; verifier tests pass.
- [ ] 2.2 [DB-SQLC] Add migration for the webhook dedup table (`stytch_webhook_events`: `event_id` unique, received_at, processed) + SQLC queries; verify the down migration. Verification: `make migrateup`/`migratedown` in the standard flow; `make sqlc` regenerates cleanly.
- [ ] 2.3 [BE-INFRA] Implement `POST /api/webhooks/stytch` (public, no session): verify signature → 401 before any work; transaction-isolated dedup on `event_id` (dedup row written in the same tx as resulting writes, replays ack 200 without effects); map recognized lifecycle events (`direct.member.create/update/delete`, `direct.organization.create/update`, `dashboard.member.*`, `scim.member.*`) → re-fetch authoritative state via the Stytch API (breaker-guarded) → append governance audit rows (`member_invited`, `member_role_changed`, `member_removed`, `organization_updated`, `member_deprovisioned`); unknown events logged + acked; breaker-open during re-fetch → log + ack, no row from unverified payload. Unit tests: happy path, replay dedup, concurrent deliveries, unknown event, breaker-open fetch, invalid signature 401 with zero side effects. Verification: `make build`; handler tests pass.
- [ ] 2.4 [OPS-GOV] Confirm the governance-event mapper writes into the existing `tipo=sistema` activity stream (same table/pattern as auth events) with bounded detail (member id, org id, role before/after, timestamp) — no credential/payload material. Verification: code review + `go test` covering row shape.

## 3. Backend — Impersonation

- [ ] 3.1 [BE-INFRA] Add `support:impersonate` permission to the Stytch RBAC policy (`admin` role, policy-driven per `stytch-authorization`) + mirror in the dev fallback role maps. Verification: policy updated in Stytch test project; fallback maps in sync; `make build`.
- [ ] 3.2 [BE-INFRA] Implement impersonation exchange: `POST /api/auth/impersonate` (authenticated, `org:manage` + `support:impersonate`, token in body) → `POST /v1/b2b/impersonation/authenticate` behind the existing breaker (breaker-open → 503 `impersonation_unavailable`) → set session cookies; reject when the target member's org ≠ impersonator's org; exit action `POST /api/auth/impersonate/exit` revokes the impersonated session. Auth middleware SHALL surface impersonation claims (`impersonating`, `impersonator_id`, `impersonator_email_address`) on the identity. Unit tests: success, org mismatch, breaker-open, missing permission, exit revoke. Verification: `make build`; `go test ./internal/modules/auth/...` passes.
- [ ] 3.3 [BE-INFRA] Record `support_impersonation_started` / `support_impersonation_ended` audit rows (best-effort, non-blocking, bounded detail) via the existing audit path. Verification: unit tests assert row shape; failure path does not block impersonation.

## 4. Backend — Device Fingerprinting

- [ ] 4.1 [DB-SQLC] Add the member device-fingerprint table/column (member id, org id, `device_id`, timestamp; no PII) + SQLC queries; verify down migration. Verification: `make migrateup`/`migratedown`; `make sqlc`.
- [ ] 4.2 [BE-INFRA] Implement `POST /api/risk/device-fingerprint` (authenticated, same-origin): persist `device_id` per member-org; return 401 unauthenticated; track volume against the 10,000/month budget with a warning log near the limit. Unit tests: persist, 401, budget warning. Verification: `make build`; tests pass.

## 5. Frontend

- [ ] 5.1 [FE-NEXT] Member profile (settings/team view): render "Impersonate" entry gated by `org:manage` + `support:impersonate`; on start, call `POST /api/auth/impersonate`; render the persistent "viewing as <member>" banner on all protected pages with exit action. Unit/component tests: gate visibility, banner presence for impersonated identity, exit flow. Verification: `pnpm lint`; `pnpm build`; component tests pass.
- [ ] 5.2 [FE-NEXT] Login surface: call `stytch.b2b.fingerprint()` at session start and POST the `device_id` to the authenticated endpoint (best-effort, failure non-blocking). Unit tests: capture + submit, failure ignored. Verification: `pnpm build`; tests pass.
- [ ] 5.3 [FE-NEXT] Confirm the `?view=audit` view renders the new governance event types without UI changes (verify the tipo filter + rows render). Verification: `pnpm build`; audit view tests pass.

## 6. Docs

- [ ] 6.1 [OPS-GOV] Update `STYTCH_CONFIGURATION.md`: impersonation enablement + support console usage + org-scope note; webhook endpoint, `whsec_` rotation procedure, event taxonomy; fingerprint 10K/mo budget; streaming destinations + free-beta note + 30-day dashboard retention. Verification: doc review.

## 7. Verification Gate

- [ ] 7.1 [BE-INFRA] `make build`; full `go test ./...` passes. Verification: exit 0.
- [ ] 7.2 [FE-NEXT] `pnpm lint`, `pnpm build`, `pnpm test` pass. Verification: exit 0.
- [ ] 7.3 [OPS-GOV] E2E in the Stytch test project: (a) impersonation — mint a token in the dashboard support console (record whether an API mint path exists), exchange, verify 60-min non-extendable session + JWT impersonation claims + audit rows; (b) webhooks — trigger a member invite/role change/SCIM deprovision and verify governance rows arrive idempotently; (c) fingerprint — capture + persist + member-profile display; (d) streaming — confirm events appear at the destination. Record outcomes (incl. `scim.member.*` event names) in this task. Verification: all four E2E flows pass; `openspec validate stytch-ops-security-suite` passes.
- [ ] 7.4 [OPS-GOV] Confirm no local credential storage: grep the diff for `whsec_`, impersonation tokens/sessions, or payload dumps in DB rows — none; audit rows bounded (code review). Verification: grep + review recorded in this task.
