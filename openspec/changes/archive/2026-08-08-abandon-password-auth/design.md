## Context

The codebase's authentication surface is exclusively magic-link based: the signup flow creates the owner member via Stytch `Members.Create`, login is handled by the `/auth` page's `sendMagicLink` server action (Stytch `loginOrSignup`), and `/authenticate` consumes the magic link token. Two OpenSpec changes — `fix-password-auth-org-id` (5/8 tasks marked done) and `add-password-auth` (0/23) — describe a password login flow that was never implemented: no `LoginRequest.OrganizationID`, no `Passwords.Authenticate` invocation, no `POST /auth/login` route exists in the codebase. The `signup-stytch-compliance` living spec explicitly bans `owner_password` from the signup payload. This change rejects that planned work and locks in the passwordless behavior with browser-level E2E tests.

## Goals / Non-Goals

**Goals:**
- Remove the two stale password-auth changes from the active set by archiving them without merging their delta specs
- Document the abandonment inside each archived change directory
- Add E2E tests that prove the auth surface stays passwordless: no password fields on `/signup` and `/auth`, no `owner_password` in the signup request payload, no password login endpoint, magic-link landing via `/authenticate`

**Non-Goals:**
- Building any password endpoint or password input
- Removing the dead `Password` field from `CreateAuthMemberRequest` (documented legacy, out of scope)
- Introducing local credential storage of any kind

## Decisions

**D1: Manual archive move instead of `openspec archive`.**
The `openspec archive` command merges delta specs into `openspec/specs/` before archiving. `fix-password-auth-org-id` carries a delta for `stytch-authorization` (password org resolution) and `add-password-auth` carries a new `password-auth` capability spec. Merging those would encode requirements for code that does not exist, contradicting `signup-stytch-compliance`. Alternative considered: marking them "rejected" in place — rejected because the archive directory is the established home for closed changes and keeps the active list clean. Archive naming follows the existing convention: `archive/2026-08-08-<name>`.

**D2: E2E tests avoid real Stytch calls.**
The `sendMagicLink` server action and `/authenticate` flow call Stytch directly and need credentials; the CI/local test environment uses mock auth (`AUTH_MOCK_ENABLED`, `X-Test-Org-ID`). The new spec therefore asserts only what is observable without real Stytch: DOM structure (no `input[type=password]`), network interception of the signup payload (no `owner_password` key), and endpoint absence (`POST /auth/login` → 404/405). Alternative considered: exercising `sendMagicLink` end-to-end with a stubbed Stytch client — rejected as requiring brittle test doubles in server actions.

**D3: Reuse existing Playwright infrastructure.**
`next_b2b_starter/e2e/playwright.config.ts` already points `testDir` at `./specs` and runs against `http://localhost:3001`; mock-auth fixtures live in `e2e/fixtures/auth.ts`. The new spec drops into `e2e/specs/` with a small `signup.page.ts` page object, matching the `add-crm-e2e-tests` pattern. No config changes required.

## Risks / Trade-offs

- [Archived changes are easy to resurrect accidentally] → `ABANDONED.md` notes state the rejection reason and point to `signup-stytch-compliance`; anyone reopening must file a new proposal
- [Signup payload interception test breaks if the payload shape changes] → The assertion checks only the *absence* of `owner_password`, not full payload equality, so it stays stable across additions
- [Future password work reappears as an ad-hoc change] → The E2E spec fails loudly if a password field or `POST /auth/login` endpoint is added without a change proposal
