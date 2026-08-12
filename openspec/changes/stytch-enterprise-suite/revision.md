# Revision Coverage — stytch-enterprise-suite (pass 1)

STATUS: COVERED
PASS: 1
VERDICT: REJECTED (council, initial)
COVERAGE: 8/8 required design changes addressed in planning artifacts; 0 deferred
DATE: 2025-07-07 (UTC)

## Verdict item 1 — SSO JIT org-policy contract (HIGH)

- **Action:** Replaced `sso_jit_provisioning: ALLOWED` with `RESTRICTED` + `sso_jit_provisioning_allowed_connections` (the org's active connection ids — least privilege, never org-wide `ALL_ALLOWED`). Removed the nonexistent org-level `sso_implicit_role_assignment`; JIT-provisioned members now stated to receive Stytch's default `stytch_member` role, with per-connection `saml_connection_implicit_role_assignment` noted as the out-of-scope path for custom roles.
- **SDK verification:** `stytch@12.43.0` `B2BOrganizationsUpdateRequest` confirms `sso_jit_provisioning ∈ [ALL_ALLOWED, RESTRICTED, NOT_ALLOWED]`, `sso_jit_provisioning_allowed_connections` exists, `sso_implicit_role_assignment` is absent, `sso_default_connection_id` exists.
- **Files:** design.md (Context, D2), specs/enterprise-sso/spec.md, tasks.md (2.1, 2.2, 7.3a).

## Verdict item 2 — SSO-JIT enablement mechanism

- **Action:** Specified: the org admin enables SSO JIT through the governed auth-policy surface (settings `?view=access` JIT card, shown when the org has ≥1 active SSO connection) writing `RESTRICTED` + connection ids via `OrgAuthPolicyUpdater`. The Admin Portal runs client-side with no backend hook and does NOT auto-write org policy. Default `NOT_ALLOWED`.
- **Files:** design.md (D2, D3), specs/enterprise-sso/spec.md ("SSO JIT provisioning per org" requirement), tasks.md (5.1, 5.2).

## Verdict item 3 — `allowed_auth_methods` semantics and enums (HIGH)

- **Action:** The updater now always writes `auth_methods: RESTRICTED` alongside `allowed_auth_methods` (enforced-list mode; `ALL_ALLOWED` default silently ignores the list otherwise). Enum values corrected to SDK values (`magic_link`, `email_otp`, `sso`, `google_oauth`, `microsoft_oauth`); dropped `email_magic_link`, generic `oauth`, `passkeys`, `m2m`. First-write preservation of existing orgs' method set specified (read org's current effective set → persist `RESTRICTED` + preserved set + additions).
- **SDK verification:** `allowed_auth_methods` docstring: "This list is enforced when `auth_methods` is set to `RESTRICTED`"; accepted values confirmed.
- **Files:** design.md (Context, D3, D7), specs/enterprise-sso/spec.md, specs/stytch-login-surface/spec.md (Email OTP requirement), tasks.md (2.1, 2.2, 5.2, 7.3f).

## Verdict item 4 — Anti-enumeration claims corrected (MEDIUM)

- **Action:** Reworded the unenforceable "no magic link SHALL be sent to the unknown address" scenarios to "SHALL surface no joinable organization / SHALL NOT reveal any organization" (discovery lists orgs only for members/invitees/JIT-eligible addresses — verified). D1 rationale reconciled: discovery sends DO reach unknown addresses, the in-process `magic-link-limiter.ts` can no longer bound login-path sends (client-side), and Stytch's per-email rate limit is the sole backstop. Posture documented in `STYTCH_CONFIGURATION.md` task (6.1).
- **Files:** design.md (D1, Risks), specs/stytch-login-surface/spec.md, specs/stytch-nextjs-components/spec.md, tasks.md (6.1).

## Verdict item 5 — Admin Portal premise verified; D4 revised (HIGH)

- **Action:** Moved from "verified premise" to `## Assumptions`/Open Questions. D4 now carries an availability gate at task 3.2: Branch A (reviewed version bump, updating the "Dependencies: none new" claim) or Branch B (Go-backed SSO/SCIM connection CRUD with custom admin forms; deltas revised if chosen). The "already ship in the pinned versions" claim is retracted in proposal.md (Impact/Dependencies) and design.md (Context).
- **SDK verification:** `AdminPortalSSOMountOptions`/`AdminPortalSCIMMountOptions`/`client.portal` and AdminPortal React components NOT found in `@stytch/vanilla-js@5.41.0` (`dist/b2b/index.d.ts` exports only `StytchB2BUIClient` with `mount`/`mountIdentityProvider`), `@stytch/nextjs@21.15.1` (`dist/b2b/index.d.ts` exports only `StytchB2B`/`B2BIdentityProvider`), or `@stytch/core@2.64.0` public types. Headless SSO/SCIM APIs confirmed present.
- **Files:** design.md (Context, Assumptions, D4, Risks, Open Questions), proposal.md (Impact), tasks.md (3.2), specs/enterprise-sso/spec.md (availability scenario).

## Verdict item 6 — Market sections + pricing verification (MEDIUM)

- **Action:** Added `## Market & Unit Economics` and `## Market Risk` to design.md (MAU headroom with SSO/SCIM/JIT members, per-project 5-connection budget vs. multi-org pipeline with the $125/conn/mo pass-through decision, discovery-send email costs, branding posture, Colombian email-abuse/locale risks). Pricing figures moved to `## Assumptions` as external claims with a dated re-verification task (new task 6.3).
- **Files:** design.md (Assumptions, Market & Unit Economics, Market Risk), tasks.md (6.3).

## Verdict item 7 — `auth-email-check` fate

- **Action:** Retained branch: the login form no longer performs the check, so requirement 2 (auth page resolves API base URL) is re-scoped to the endpoint's remaining callers. **New MODIFIED delta created:** `specs/auth-email-check/spec.md` (new file under the specs artifact, verdict-mandated — the pipeline's `openspec validate` will check it against the living spec). Requirement 1 (endpoint contract) unchanged; if the task-4.2 caller audit deletes the endpoint, requirement 1 is removed via a follow-up spec change (recorded in tasks 4.2).
- **Files:** specs/auth-email-check/spec.md (NEW), design.md (D1, Open Questions), tasks.md (4.2, 6.2).

## Verdict item 8 — Minor contract cleanups in one pass

- **Action:** Context `email_jit_provisioning` value list corrected to `[RESTRICTED, NOT_ALLOWED]`; `allowed_auth_methods` list corrected (item 3); OAuth-JIT implication of domain JIT stated (design D2, `stytch-login-surface` delta, task 5.1 UI copy); read-path breaker→503 mapping added (`auth_policy_unavailable`, D3, task 2.2/2.3/5.1); Spanish `locale: "es"` + strings pinned for `StytchB2B` (task 4.1, D8) and Admin Portal mounts (task 3.2); `ValidateAuthPolicyUpdate` now covers `sso_default_connection_id` org-ownership (D3, task 2.1).
- **Files:** design.md (Context, D3, D8, Open Questions), specs/stytch-login-surface/spec.md, tasks.md (2.1, 3.2, 4.1, 5.1).

## Residual / deferred

- None of the 8 items deferred. Non-blocking notes carried in design `## Open Questions`: sendMagicLink/check-email caller audit outcome (task 4.2 — may still delete the endpoint, requiring a follow-up spec change); SCIM group→role default confirmed at E2E; Admin Portal Branch A/B decision recorded at task 3.2; pricing drift reconciled at task 6.3.
- `crm-frontend` and `scim-provisioning` deltas left as the required end state (Admin Portal surface); the availability gate and fallback (which would revise them) live in design D4 / task 3.2.
