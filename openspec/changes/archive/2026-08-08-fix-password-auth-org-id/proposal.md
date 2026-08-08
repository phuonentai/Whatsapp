## Why

The password-based login flow returns `organization_not_found` from Stytch because the `Authenticate` call passes an empty `OrganizationID`. The B2B password authenticate API requires the organization ID to route the authentication to the correct org. Without it, Stytch cannot locate the member's organization, and users cannot log in with their password after signup.

## What Changes

- Fix the `Login` service method to resolve the member's Stytch organization ID from the local database before calling `Passwords.Authenticate`
- Update `Authenticate` on the repository to accept and forward the organization ID to Stytch

## Capabilities

### New Capabilities

None — this is a bug fix within the existing `stytch-authorization` capability.

### Modified Capabilities

- `stytch-authorization`: The `Login` flow must resolve the member's organization before authenticating the password

## Impact

- `go-b2b-starter/internal/modules/organizations/app/services/member_service_impl.go` — `Login()` method: add org resolution step before calling repo `Authenticate`
- `go-b2b-starter/internal/modules/organizations/infra/repositories/stytch_member_repository.go` — `Authenticate()`: populate `OrganizationID` from the resolved value
- `go-b2b-starter/internal/modules/organizations/domain/auth_provider.go` — `LoginRequest`: add `OrganizationID` field