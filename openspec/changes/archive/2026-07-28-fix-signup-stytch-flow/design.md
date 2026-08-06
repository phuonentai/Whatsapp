## Context

The current signup bootstrap flow in `member_service_impl.go` performs a multi-step sagas operation with rollback. The relevant steps for this change are:

```
Step 1: Create org in Stytch (Organizations.Create)
Step 2: Create local org in PostgreSQL
Step 3: Create member in Stytch (Members.Create, SendInvite: false)
Step 4: Assign admin role in Stytch (Members.Update with roles)
Step 5: Create local account in PostgreSQL
Step 6: Send magic link (MagicLinks.Email.LoginOrSignup) — AFTER shouldRollback = false
```

Deviations from Stytch guidelines:
- Step 3 creates a member without an invite, then Step 6 sends a separate LoginOrSignup — Stytch's `CreateMember` natively supports `SendInvite: true` which sends a Stytch-branded invite email with a magic link
- Step 6 errors are silently logged but the response reports `MagicLinkSent: true`
- The Stytch Go SDK is v16; current is v18
- The frontend generates and sends an `owner_password` that the backend discards

## Goals / Non-Goals

**Goals:**
1. Replace the two-step member-create + magic-link with a single `CreateMember(SendInvite: true)` call
2. Surface magic link / invite errors to the caller instead of swallowing them
3. Remove `owner_password` from the frontend DTO and signup payload
4. Upgrade stytch-go from v16 to v18

**Non-Goals:**
- Changing the overall bootstrap architecture (rollback stack, step ordering beyond what's needed for the fix)
- Switching to the Discovery authentication flow (separate change)
- Adding PKCE support for magic links (security hardening, deferred)
- Changing the frontend Stytch SDK initialization or the `/authenticate` callback page
- Upgrading the frontend Stytch SDK (`@stytch/nextjs/b2b`)

## Decisions

### Decision 1: Replace SendMagicLink + LoginOrSignup with CreateMember(SendInvite: true)

The current flow creates a member in Stytch with `SendInvite: false`, then separately calls `MagicLinks.Email.LoginOrSignup`. These are logically equivalent — both result in Stytch sending a magic link email to the member.

**Change:** Set `SendInvite: true` on the `CreateMember` call and remove the separate `LoginOrSignup` step entirely.

After member creation, Stytch sends an invite email containing a magic link. When the user clicks it, the existing `/authenticate` callback (in Next.js) exchanges the token via `magicLinks.authenticate()` — same as current behavior.

**Bootstrap flow after change:**
```
Step 1: Create org in Stytch (Organizations.Create)
Step 2: Create local org in PostgreSQL
Step 3: Create member in Stytch (Members.Create, SendInvite: true)
        → Stytch sends invite magic link natively
Step 4: Assign admin role in Stytch
Step 5: Create local account in PostgreSQL
        → shouldRollback = false
Step 6: (REMOVED — was LoginOrSignup)
```

**Alternative considered:** Keep the two-step approach but move the magic link before `shouldRollback`. Rejected because `CreateMember(SendInvite: true)` is Stytch's intended API for this exact scenario — no need for a separate call.

**Alternative considered:** Use Stytch's Discovery flow for signup. Rejected as a larger architectural change beyond this fix.

### Decision 2: Remove the `owner_password` field

**Frontend:** Remove `owner_password` from `SignupMagicLinkRequestDto` and stop generating it. Remove `password-generator.ts` if it has no other consumers.

**Backend:** No changes needed — `ShouldBindJSON` already ignores the field. But verify that no other consumers depend on it in the response path.

**Alternative considered:** Pass the password to Stytch's `CreateMember` (which accepts a password field). Rejected because the bootstrapped org uses passwordless magic link auth — no password is needed.

### Decision 3: Stytch SDK upgrade strategy

Upgrade `stytch-go/v16` → `stytch-go/v18` in `go.mod`. Run `go mod tidy` to update dependencies. The v16→v18 migration path involves:

1. `LoginOrSignup` → This endpoint is removed in v18 for the b2b magiclinks email package. The v18 Go SDK uses `discovery.Send` and org-specific `loginOrSignup` differently. **This reinforces Decision 1** — removing the LoginOrSignup call eliminates the need to deal with this API change.

2. API package paths may have changed between v16 and v18. Verify imports in all Stytch repository files.

**Rollback:** If the SDK upgrade introduces unforeseen issues, revert to v16 and keep the v16-compatible member creation code as a fallback branch.

### Decision 4: Structured error mapping

Add error code mapping in the handler so Stytch API errors produce actionable diagnostics:

```go
type SignupErrorCode string

const (
    ErrCodeStytchUnauthorized SignupErrorCode = "STYTCH_UNAUTHORIZED"
    ErrCodeStytchUnreachable  SignupErrorCode = "STYTCH_UNREACHABLE"
    ErrCodeSlugConflict       SignupErrorCode = "SLUG_CONFLICT"
    ErrCodeDBConnection       SignupErrorCode = "DB_CONNECTION_FAILED"
    ErrCodeInviteFailed       SignupErrorCode = "INVITE_FAILED"
)
```

This is additive — the existing `response.Error()` signature can accommodate an optional code parameter.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| `CreateMember(SendInvite: true)` sends an "invite" email (not "login" email) which may have different Stytch dashboard template or email copy | Verify the Stytch dashboard invite template is configured. Users may want to customize the invite email text in the Stytch dashboard. |
| Stytch SDK v18 import paths changed; existing Stytch integrations (org, member, RBAC) may need import updates | Test all Stytch API calls post-upgrade. Keep a v16 fallback if needed. |
| Removing `owner_password` breaks other flows that depend on it | Search for all usages of `owner_password` and `password-generator.ts` before removal. |
| Changing the invite flow means Stytch sends the magic link at Step 3 (earlier), so if Steps 4-5 fail, the user may receive an invite email for an incomplete org setup | The rollback stack includes member deletion, which should clean up the Stytch-created member. Confirm Stytch sends a second email on transient failures (e.g., a failed rollback that doesn't delete the member). |
| `LoginOrSignup` removal breaks the `SendMagicLink` method on `AuthMemberRepository` which may be called elsewhere | Audit all callers of `SendMagicLink`. Update or remove accordingly. |
