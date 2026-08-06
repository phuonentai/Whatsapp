## Why

The signup flow deviates from Stytch's recommended patterns in five ways: it makes redundant API calls (creating a member then separately sending a magic link), silently swallows magic link errors (reporting `MagicLinkSent: true` on failure), sends a dead-weight `owner_password` field that the backend discards, runs on Stytch SDK v16 (two major versions behind), and triggers Stytch frontend SDK cache-clear errors. The result is users getting 500 errors on signup with no actionable diagnostics.

## What Changes

- Remove the separate `LoginOrSignup` call after member creation — switch to `SendInvite: true` on the `CreateMember` call so Stytch handles the magic link invite email natively
- Fix the `MagicLinkSent` false-positive: move the send step before `shouldRollback = false` and surface errors to the caller
- Remove the `owner_password` field from the frontend signup DTO and payload — it is generated, transmitted, then silently discarded by the backend
- Upgrade `stytch-go` from v16 to v18 to stay current with Stytch's API improvements and avoid potential API breakage
- Add structured error diagnostics to the signup handler so 500 errors include a machine-readable error code and actionable detail

## Capabilities

### New Capabilities

- `signup-stytch-compliance`: Align the organization bootstrap flow with Stytch's recommended invite-and-magic-link pattern and add structured error reporting

### Modified Capabilities

- `stytch-authorization`: Update the bootstrap service to use Stytch's native member invite flow instead of a separate LoginOrSignup after creation

## Impact

- `go-b2b-starter/internal/modules/organizations/app/services/member_service_impl.go` — restructure bootstrap steps 3-6
- `go-b2b-starter/internal/modules/organizations/infra/repositories/stytch_member_repository.go` — may simplify SendMagicLink method if it's only used for the removed LoginOrSignup path
- `go-b2b-starter/internal/modules/organizations/infra/repositories/stytch_organization_repository.go` — no direct changes, but callers change
- `go-b2b-starter/internal/modules/organizations/member_handler.go` — add structured error mapping
- `go-b2b-starter/go.mod` — upgrade stytch-go v16 → v18
- `next_b2b_starter/lib/api/api/dto/auth.dto.ts` — remove `owner_password` field from `SignupMagicLinkRequestDto`
- `next_b2b_starter/lib/api/api/repositories/signup-repository.ts` — stop generating and sending `owner_password`
- `next_b2b_starter/lib/utils/password-generator.ts` — may be removable if no other consumer exists
