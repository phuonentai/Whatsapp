## Context

The current signup flow uses Stytch's `MagicLinks.Email.Invite` to create a member and send an invite email with a magic link. The login flow uses `MagicLinks.Email.LoginOrSignup`. The Go SDK v18 exposes `PasswordsClient` with `Authenticate` and `Migrate` methods, but does not have a `Passwords.Create` wrapper — that endpoint exists in the Stytch REST API (`POST /v1/b2b/passwords`) but must be called via raw HTTP.

A password column exists on the domain `CreateAuthMemberRequest` but is dead code — never populated or sent.

## Goals / Non-Goals

**Goals:**
1. Add password field to the signup form and DTO
2. On signup, create the Stytch member without invite and set the password via `POST /v1/b2b/passwords`
3. Add a password-based login endpoint (`POST /api/auth/login`) using Stytch `Passwords.Authenticate`
4. Replace frontend magic-link login form with email + password form

**Non-Goals:**
- Removing the existing magic link infrastructure entirely (keep for optional use)
- Password reset flow (separate change)
- Multi-factor authentication (MFA)
- OAuth/SSO integration

## Decisions

### Decision 1: Set password via raw Stytch REST API call

The Go SDK's `PasswordsClient` does not have a `Create` method. The Stytch REST API endpoint `POST /v1/b2b/passwords` accepts `{ organization_id, member_id, password }` and creates a password for the member. Call it directly via `net/http`.

Flow:
```
Signup:
  Step 1: Create Stytch org (Organizations.Create)
  Step 2: Create local org
  Step 3: Create Stytch member (Organizations.Members.Create, no invite)
  Step 4: POST /v1/b2b/passwords → set password for member
  Step 5: Assign admin role
  Step 6: Create local account
```

**Alternative considered:** Migrate the password hash via `Passwords.Migrate`. Rejected because it requires hashing the password server-side with a specific algorithm; the raw API call is simpler and lets Stytch handle hashing.

**Alternative considered:** Frontend-only password setting via Stytch frontend SDK. Rejected because it requires an intermediate session token flow and adds complexity.

### Decision 2: Add login endpoint that calls Passwords.Authenticate

```
POST /api/auth/login
Body: { email, password }
→ Backend: determine org via email lookup
→ Stytch: Passwords.Authenticate({ organization_id, email, password, session_duration_minutes })
→ Response: { session_token, session_jwt, member }
```

The existing `GET /api/auth/check-email` endpoint can be reused to find which org the member belongs to.

### Decision 3: Frontend — replace magic link login with password form

The `/auth` page currently collects only email and sends a magic link. Change it to also collect a password and call the new `POST /api/auth/login` endpoint. Keep the "send magic link" option as a toggle for users who prefer it.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| Stytch `POST /v1/b2b/passwords` endpoint may behave differently in test vs live environment | Test thoroughly in test environment before deploying to live |
| Raw HTTP call duplicates error handling patterns (no SDK wrapper) | Wrap in a dedicated `createPassword` method in the repository layer with consistent error mapping |
| Password validation (strength, length) must happen client-side or via `Passwords.StrengthCheck` | Add `StrengthCheck` call during signup for feedback, enforce minimum length on both sides |
| Existing users without passwords can't use the login form | The magic link option remains available as a fallback |
