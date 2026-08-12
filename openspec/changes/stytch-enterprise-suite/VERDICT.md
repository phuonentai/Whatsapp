# Council Verdict — stytch-enterprise-suite (pass 1 re-review)

STATUS: APPROVED
MARKET: PASS

Re-review of the REVISED design after the pass-0 REJECTED verdict (8 required design changes). All 8 items were verified addressed in the planning artifacts (design.md, proposal.md, tasks.md, and all six delta specs), with the SDK/contract claims independently re-checked against the installed packages for this pass.

## Premise verification notes (facts checked against the codebase and pinned SDKs in this pass)

- **Installed SDK versions match the design's claims:** `@stytch/vanilla-js@5.41.0`, `@stytch/nextjs@21.15.1`, `stytch@12.43.0` (JS), `@stytch/core@2.64.0` (pnpm store), Go `stytchauth/stytch-go/v18@v18.1.0` (go.mod).
- **Admin Portal absence confirmed (verdict item 5):** `@stytch/vanilla-js@5.41.0` `dist/b2b/index.d.ts` exports only `StytchB2BUIClient` with `mount`/`mountIdentityProvider`; `@stytch/nextjs@21.15.1` `dist/b2b/index.d.ts` exports only `StytchB2B`/`B2BIdentityProvider`. No `client.portal`, `AdminPortalSSOMountOptions`, or `AdminPortalSCIMMountOptions` anywhere in the type surface. The revision's move of this from "verified premise" to Assumptions/Open Questions + the task-3.2 decision gate (Branch A bump / Branch B Go-backed fallback) is **correct**; proposal.md Impact now states the "Dependencies: none new" claim is NOT verified.
- **Org-update contract verified in BOTH `stytch-go/v18 v18.1.0` (`organizations.UpdateParams`) and `stytch@12.43.0` (`B2BOrganizationsUpdateRequest`):**
  - `sso_jit_provisioning` accepted values are `ALL_ALLOWED | RESTRICTED | NOT_ALLOWED` — there is NO `ALLOWED` value (item 1 fixed: design uses `RESTRICTED` + `sso_jit_provisioning_allowed_connections`).
  - `sso_jit_provisioning_allowed_connections` and `sso_default_connection_id` exist; **no org-level `sso_implicit_role_assignment` field exists** anywhere in the update contract (removed from the design; JIT default `stytch_member` stated).
  - `email_jit_provisioning` accepted values are `RESTRICTED | NOT_ALLOWED`, documented as governing provisioning via **Email Magic Link or OAuth** (item 8's OAuth-JIT implication is accurate).
  - `auth_methods` accepted values are `ALL_ALLOWED | RESTRICTED`; `allowed_auth_methods` docstring states *"This list is enforced when `auth_methods` is set to `RESTRICTED`"* and the accepted values are `sso, magic_link, email_otp, password, google_oauth, microsoft_oauth, slack_oauth, github_oauth, hubspot_oauth` (items 3/8 fixed: updater writes `RESTRICTED` + complete list, SDK-accurate enums, `email_magic_link`/generic `oauth`/`passkeys`/`m2m` dropped).
  - `email_allowed_domains` is enforced when **either** `email_invites` or `email_jit_provisioning` is `RESTRICTED` — relevant to the residual note in 1-DBA below.
- **Discovery send contract (item 4):** `B2BMagicLinksEmailDiscoverySendRequest` requires only `email_address` plus optional redirect/pkce/template/locale/expiration — no membership-check or suppression hook exists, confirming D1's honest posture change. `locale` values verified `en | es | pt-br | fr` (supports D8). The in-repo `STYTCH_CONFIGURATION.md` states the custom flow exists because Stytch's UI "always sends emails" — the retraction of the "bounded without custom code" claim is honest and now consistent across D1, Risks, and the deltas.
- **Codebase premises:** `internal/modules/organizations/domain/mfa_policy.go` is the shipped `MfaPolicyUpdater` + `ValidateMfaPolicyUpdate` template (typed enums, breaker-guarded contract, 503 mapping) — D3 mirrors it faithfully; `settings-content.tsx` implements the `?view=` allowlist with per-view `org:manage` gates (D5 premise TRUE); the `crm-frontend` (AdminPortal mandate) ⇄ `stytch-nextjs-components` (custom flow) contradiction in the living specs is real and the MODIFIED deltas reconcile it in one coherent direction.
- **Delta coherence:** `stytch-login-surface` (ADDED), `enterprise-sso` (ADDED, includes the Admin-Portal availability-gate scenario and an implementable SSO-JIT enablement trigger), `scim-provisioning` (ADDED), `stytch-nextjs-components` (MODIFIED, requirement headers match living spec), `crm-frontend` (MODIFIED, headers match), `auth-email-check` (MODIFIED, header matches; requirement 1 contract preserved, requirement 2 re-scoped to remaining callers per item 7). Task 4.2 records the caller audit and the conditional follow-up spec change.
- **Pricing figures** (10K MAU, 5 connections, $125/conn/mo, Audit Logs $799/mo, branding $99/mo) remain external claims, now correctly in `## Assumptions` with a dated re-verification gate at task 6.3 (item 6).

## Per-persona findings

### 1. Staff Security Engineer

- **[RESOLVED-HIGH] SSO JIT org-policy contract (item 1):** `sso_jit_provisioning: RESTRICTED` + `sso_jit_provisioning_allowed_connections` (org's active connection ids, never org-wide `ALL_ALLOWED`) is the value the SDK actually accepts; phantom org-level `sso_implicit_role_assignment` removed; JIT-provisioned members stated to receive Stytch's default `stytch_member` role with per-connection `saml_connection_implicit_role_assignment` named as the out-of-scope path for custom roles. Enterprise-sso delta scenarios cover enable/disable/default/least-privilege.
- **[RESOLVED-HIGH] `allowed_auth_methods` semantics (item 3):** updater always writes `auth_methods: RESTRICTED` alongside the complete `allowed_auth_methods` list; enums SDK-accurate; first-write preservation of existing orgs' method set is specified, unit-tested (2.2) and E2E-verified (7.3f).
- **[RESOLVED-MEDIUM] Anti-enumeration posture (item 4):** unenforceable "no send to unknown address" scenarios replaced with "SHALL surface no joinable organization / SHALL NOT reveal any organization" (achievable per the discovery contract); the limiter's loss on the login path and Stytch's per-email rate limit as sole backstop are documented in D1, Risks, and the task-6.1 `STYTCH_CONFIGURATION.md` update.
- **[RESOLVED-LOW] SSO-JIT enablement mechanism (item 2):** the trigger is now explicit — the org admin toggles SSO JIT through the governed `?view=access` card (shown when ≥1 active SSO connection), which writes `RESTRICTED` + connection ids via `OrgAuthPolicyUpdater`; the Admin Portal does not auto-write org policy (client-side, no backend hook). Verifiable at E2E (7.3a).
- **[RESOLVED-LOW] `ValidateAuthPolicyUpdate` (item 8):** now covers `sso_default_connection_id` org-ownership (D3, task 2.1).
- **[LOW — residual, non-blocking] `email_allowed_domains` coupling:** because `email_allowed_domains` is enforced when either `email_invites` OR `email_jit_provisioning` is `RESTRICTED`, and the current tenant posture sets `email_invites: RESTRICTED` (per `STYTCH_CONFIGURATION.md`), the JIT toggle's domain-list write also constrains invite domains for those orgs. D2 states the OAuth-JIT implication but not this invite-domain coupling; the settings UI copy (task 5.1) should state that the domain list applies to invites as well while invites are domain-restricted. Behavior is consistent with the SDK contract — a copy/docs clarification, not a design defect.
- **[OK]** No local credential/secret storage; breaker-guarded writes/reads with 503 structured errors on both paths; `org:manage` gating; display-only mirror invariant stated in D3 and the deltas.

### 2. Staff DBA

- **[CLEAN] No schema changes, no migrations, no new tables.** Auth-policy mirror read from the Stytch org object, display-only; consistent with the `mfa-totp-enrollment` precedent.
- **[LOW — residual, non-blocking] First-write preservation semantics:** D3/deltas describe the preserved set for an `ALL_ALLOWED` org both as "the project-enabled methods relevant to the org — at minimum `magic_link`, plus `sso` when active connections" (design) and "at minimum `magic_link`" (delta/task). Pin down in task 2.2 whether the preserved set is the full effective (project-enabled) set or the conservative subset, so the unit test asserts the intended body exactly.
- **[OK]** The display-only invariant sentence is now carried in the `stytch-login-surface` delta as requested.

### 3. Staff SRE

- **[RESOLVED-MEDIUM] Admin Portal availability (item 5):** re-verified this pass — the exports are genuinely absent from the pinned type surface; the design now treats this as an open assumption with a task-3.2 gate (reviewed version bump that updates the "Dependencies: none new" claim, or the Go-backed fallback that revises the deltas before FE sizing). The `enterprise-sso` delta carries the gate as a scenario; D4 documents Branch B's endpoints (`GET/POST/DELETE /api/organizations/sso-connections`, `scim-connections`) behind the shared breaker.
- **[RESOLVED-LOW] Read-path breaker behavior (item 8):** `GET /api/organizations/auth-policy` maps breaker-open/unreachable to the 503 structured-error family (`auth_policy_unavailable`), mirrored in D3, tasks 2.2/2.3, and the task-5.1 UI error states.
- **[LOW — residual, non-blocking] External "~1 req/sec/email" rate-limit figure:** Stytch-platform behavior, marked approximate; record it alongside the dated pricing re-verification at task 6.3 if it should be pinned to a source. Does not affect any contract.
- **[OK]** Git + Stytch-tenant-state rollback defined (JIT → `NOT_ALLOWED`, `auth_methods` → prior set, connections deleted via portal/API); billing-verification gated OPS-GOV (task 1.1); E2E scoped to the test project before prod.

### 4. Product / GTM

- **[RESOLVED-MEDIUM] Market sections (item 6):** `## Market & Unit Economics` and `## Market Risk` are present and coherent — MAU headroom with SSO/SCIM/JIT members (with the 10K-cap watch item), per-project 5-connection budget vs. the multi-org enterprise pipeline with an explicit **pass-through** decision ($125/conn/mo absorbed into Enterprise ACV, no soft cap, no code gate, revisited at task 6.3), discovery-send email volume, branding posture, and the Colombian email-abuse/locale risks. Pricing figures correctly demoted to `## Assumptions` with a dated re-verification task.
- **[OK]** The enterprise procurement narrative (SSO gate, SCIM lifecycle ask, OAuth/OTP friction reduction) holds; $0 marginal cost posture is coherent within the free tier.

### 5. Colombia IT-Market

- **[RESOLVED-MEDIUM] Market-read gate:** cleared — unit economics and market risk sections cover the Colombia-specific asks (MAU per Colombian enterprise customer, SMB/enterprise connection-mix, discovery-send email-bombing/spam-reputation surface, Stytch-branded email acceptance posture).
- **[RESOLVED-LOW] Spanish locale (item 8):** `locale: "es"` + strings overrides pinned for `StytchB2B` (task 4.1) and the Admin Portal mounts (task 3.2); SDK locale values verified (`en | es | pt-br | fr`); Playwright `qa/` verification included in the FE tasks.
- **[RESOLVED-LOW] Context contract values (item 8):** `email_jit_provisioning ∈ [RESTRICTED, NOT_ALLOWED]` and `allowed_auth_methods` list now match the pinned SDKs exactly; the OAuth-JIT implication of domain JIT is stated in D2, the delta, and the task-5.1 UI copy.
- **[LOW — residual, non-blocking] `crm-frontend` delta phrasing:** the modified `crm-frontend` requirement names SSO/SCIM surfaces as React components (`<AdminPortalSSO />`, `<AdminPortalSCIM />`) while D4 and the `enterprise-sso`/`scim-provisioning` deltas use mount options (`AdminPortalSSOMountOptions`/`AdminPortalSCIMMountOptions`). The task-3.2 gate decides the exact shape; align the `crm-frontend` text to the chosen surface when the gate lands (Branch B already mandates revising the deltas). Cosmetic — the reconciliation direction is unambiguous.

## Required design changes

None — all 8 items from the pass-0 verdict are resolved with verified SDK contracts. Residual notes above are LOW, non-blocking clarifications for the apply stage (task 2.2 first-write semantics, task 5.1 UI copy for the invite-domain coupling, task 3.2 delta phrasing alignment, task 6.3 external-figure pinning).

## Archive decision note

Not applicable at this stage — this is the council re-review gate; the change proceeds to `/sdet` (opsx-apply) with the verified design. Archive decision is recorded in tasks.md by the apply workflow per the lifecycle gates.
