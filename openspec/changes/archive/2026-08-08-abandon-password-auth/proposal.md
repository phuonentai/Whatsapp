## Why

Two password-authentication changes — `fix-password-auth-org-id` and `add-password-auth` — claim or plan a `POST /auth/login` password flow that was never implemented: there is no `LoginRequest.OrganizationID`, no `Passwords.Authenticate` call, and no `/auth/login` route in the codebase. The plan also contradicts the living `signup-stytch-compliance` spec, which bans `owner_password` from the signup payload. Keeping these changes active invites future work against a product decision (magic-link-only authentication) and a spec contradiction.

## What Changes

- Move `openspec/changes/fix-password-auth-org-id/` and `openspec/changes/add-password-auth/` to `openspec/changes/archive/` (manual move — do **not** run `openspec archive`, which would merge their deltas into main specs; the code was never written)
- Add an `ABANDONED.md` note in each archived change directory explaining why and when it was abandoned
- Add a new E2E capability `auth-passwordless-e2e` with a Playwright spec locking in the passwordless behavior: no password fields on `/signup` or `/auth`, no `owner_password` in the signup payload, no password login endpoint, and magic-link landing via `/authenticate`

## Capabilities

### New Capabilities

- `auth-passwordless-e2e`: Browser-level E2E tests verifying the authentication surface is strictly magic-link based (no password inputs, no `owner_password` in signup payload, no `POST /auth/login` endpoint)

### Modified Capabilities

<!-- No existing capability changes its requirements — the archived password changes are rejected, and the living `signup-stytch-compliance` spec already prohibits `owner_password`. -->

## Impact

- **OpenSpec**: Two active changes moved to `openspec/changes/archive/` with `ABANDONED.md` notes; no delta specs merged into `openspec/specs/`
- **Frontend**: New `next_b2b_starter/e2e/specs/auth-passwordless.spec.ts` and `next_b2b_starter/e2e/page-objects/signup.page.ts`; existing `playwright.config.ts` and mock-auth fixtures reused unchanged
- **Backend**: No code changes
- **Rollback**: Restore the two change directories from `archive/` to `openspec/changes/` via `git checkout`; delete the E2E spec file and page object. Stytch tenant state is unaffected (no API contract or policy changes are made)
- **Non-Goals**: Not building any password endpoint. Not deleting the dead `Password` field on `CreateAuthMemberRequest` (kept as documented legacy). No local credential storage is introduced or enabled — authentication remains exclusively Stytch-managed magic links
