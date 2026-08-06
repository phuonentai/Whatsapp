## Why

The current signup flow relies on Stytch magic link emails for authentication, which adds email delivery dependency, delays time-to-first-login, and excludes users who prefer or require password-based credentials.

## What Changes

- Replace `MagicLinks.Email.Invite` on signup with Stytch's `Passwords.Create` — the user sets a password during registration instead of receiving an invite email
- Add a password-based login endpoint (`POST /api/auth/login`) that authenticates via Stytch `Passwords.Authenticate` and returns a session
- Add password field to the frontend signup form and replace the magic-link login form with an email + password form
- Remove the `sendMagicLink` server action and `/authenticate` callback page (or keep them for optional magic link fallback)

## Capabilities

### New Capabilities

- `password-auth`: Email + password authentication flow using Stytch B2B Passwords API

### Modified Capabilities

<!-- No existing specs to modify — this is a new authentication path. -->

## Impact

- `go-b2b-starter/internal/modules/organizations/app/services/member_service_impl.go` — replace `InviteMember` with password-based member creation
- `go-b2b-starter/internal/modules/organizations/infra/repositories/stytch_member_repository.go` — add `CreateMemberWithPassword` method calling Stytch `Passwords.Create`
- `go-b2b-starter/internal/modules/organizations/member_handler.go` — add login handler endpoint
- `go-b2b-starter/internal/modules/organizations/domain/auth_provider.go` — the existing `Password` field becomes active (was dead code)
- `next_b2b_starter/app/signup/page.tsx` — add password input to signup form
- `next_b2b_starter/app/auth/page.tsx` — replace magic link form with password login form
- `next_b2b_starter/lib/api/api/dto/auth.dto.ts` — add password fields, use `LoginRequestDto`
- `next_b2b_starter/lib/api/api/repositories/signup-repository.ts` — send password in payload
