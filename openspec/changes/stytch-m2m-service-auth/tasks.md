# Stytch M2M Service Auth — Tasks

## 1. Stytch Project Configuration

- [ ] 1.1 [OPS-GOV] In the Stytch test project: create the platform M2M client (`m2m.platform`) with scopes (`crm:read`, `whatsapp:send`) and the `org_ids` custom claim; confirm token issuance via `POST /v1/b2b/m2m/token/get-access-token` and record the JWT claim shape (audience value, scopes casing, custom-claims embedding) in this task. Verification: token issues; claims recorded; 1K-client free budget documented.
- [ ] 1.2 [OPS-GOV] Confirm the M2M JWT is signed by the project JWKS (same key set as member JWTs or a distinct audience) — record whether the existing JWKS cache URL covers M2M tokens or a second URL is needed. Verification: signature verification against the project JWKS succeeds; finding recorded in this task.

## 2. Backend — M2M Verification

- [ ] 2.1 [BE-INFRA] Implement `internal/platform/m2m`: two-tier verifier (JWKS fast path with ≤300s cache reuse + issuer/audience/expiry checks; slow path `POST /v1/b2b/m2m/token/authenticate-access-token` behind the existing breaker, breaker-open → 503 `m2m_auth_unavailable`). Unit tests: valid local verify, unknown key → API fallback, invalid/expired/wrong-audience → 401, breaker-open → 503. Verification: `make build`; `go test ./internal/platform/m2m/...` passes.
- [ ] 2.2 [BE-INFRA] Implement the scope→permission table (`m2m_scope_permissions`, declarative, following the `resource:action` convention) and the M2M middleware: resolve the service principal's permission set, validate `X-Stytch-Organization-Id` against the `org_ids` allowlist claim (missing/outside → 403), set identity + `OrganizationID` via the `authcontext` seam. Unit tests: scope mapping, unknown scope denied, org allowlist 403s, existing permission gates work for M2M callers. Verification: `make build`; middleware tests pass.
- [ ] 2.3 [BE-INFRA] Wire the middleware into the auth module route plumbing so it coexists with member-session auth (`X-Forwarded-Auth`/JWT) — additive, no changes to member paths. Audit log line for M2M calls (client id, org id, scope, timestamp, outcome — no token/secret material). Verification: `make build`; `go test ./internal/modules/auth/...` passes.

## 3. First Consumer Wiring

- [ ] 3.1 [BE-INFRA] Identify the concrete out-of-band caller (candidate: scheduled campaign delivery per `whatsapp-campaigns` launch lifecycle). If a caller exists, wire it to call the protected send surface with the M2M client credentials from env; if none exists, record the gated status in this task (capability ships with middleware + tests + provisioned client). Verification: caller identified or gated status recorded; wired path passes unit tests with mocked token issuance.

## 4. Docs

- [ ] 4.1 [OPS-GOV] Update `STYTCH_CONFIGURATION.md`: M2M client provisioning steps, 1K-client free budget, scope naming convention, `org_ids` allowlist requirement, secret rotation procedure. Verification: doc review.

## 5. Verification Gate

- [ ] 5.1 [BE-INFRA] `make build`; full `go test ./...` passes. Verification: exit 0.
- [ ] 5.2 [OPS-GOV] E2E in the Stytch test project: issue a token for `m2m.platform`, call a protected endpoint with the M2M JWT + an allowlisted org header → 200; non-allowlisted org → 403; missing header → 403; invalid token → 401; breaker-open → 503. Record outcomes (incl. JWKS URL/audience finding from 1.2) in this task. Verification: all cases pass; `openspec validate stytch-m2m-service-auth` passes.
- [ ] 5.3 [OPS-GOV] Confirm no local credential storage: grep the diff for client secrets/token persistence — none; audit lines bounded (code review). Verification: grep + review recorded in this task.
