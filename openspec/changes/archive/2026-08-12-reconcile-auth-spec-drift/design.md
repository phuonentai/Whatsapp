# Reconcile Auth Spec Drift — Design

## Context

Spec-vs-code drift identified: `stytch-nextjs-components` (living spec) mandates `<StytchB2B />` on `/login` and AdminPortal components on `/settings`; the implementation (deliberately, per `STYTCH_CONFIGURATION.md`) uses custom forms on `/auth` and custom member management. The pre-built Stytch B2B component always sends a magic link on email submit, which contradicts the anti-abuse design (validate membership first, neutral failure, JIT provisioning disabled, discovery restricted).

## Decisions

### D1 — MODIFIED requirements (custom flow becomes the spec)

- **Login page (renamed requirement):** "Login page renders a custom email form" — `/auth` SHALL render an email-only form (no password), SHALL validate membership via Stytch `members.search` before sending, SHALL return the neutral message for unknown emails, and SHALL NOT call `magicLinks.email.loginOrSignup` for non-members. Scenarios: known member → link sent; unknown member → neutral message, no email; no password input anywhere.
- **Member management (renamed requirement):** "Settings uses custom member management" — `/settings` SHALL render the custom member list/invite components (`member-list.tsx`, `invite-member.tsx`) and SHALL NOT depend on Stytch AdminPortal components. Scenario: admin invites member → `SendInvite` flow → member appears in list.

### D2 — REMOVED requirements

- Remove "Login page renders pre-built Stytch B2B component" (Discovery/SSO component mandates) and "No custom form components in auth pages" — replaced by D1 requirements that match the design.
- Keep the SSO product as an explicit future requirement note in the Purpose (not a hard current requirement): SSO surfacing is deferred until SSO connections are productized.

### D3 — Docs alignment

- `STYTCH_CONFIGURATION.md`: replace `/api/auth/magic-link` endpoint references with the `sendMagicLink` server action; replace `/api/auth/logout` references with the `logout` server action; keep the security rationale (anti-enumeration, JIT blocked). Session-duration value corrected by `session-lifetime-hardening` (referenced, not duplicated).

## Stytch Boundary

- No Stytch API contract changes; the spec now accurately describes the existing Stytch calls (`members.search`, `magicLinks.email.loginOrSignup` for known members only, `SendInvite`).

## Security Invariants

- The spec continues to forbid password inputs in sign-in/signup (alignment with `auth-passwordless-e2e` and `signup-stytch-compliance`).
- Anti-enumeration invariant is now spec'd: identical neutral responses for existing/non-existing members.

## Testing Strategy

- Governance validation: `openspec validate reconcile-auth-spec-drift` passes; `openspec status` shows the change complete.
- Consistency check: grep living spec `stytch-nextjs-components` for removed component mandates (none remain); `STYTCH_CONFIGURATION.md` route references point to server actions.
