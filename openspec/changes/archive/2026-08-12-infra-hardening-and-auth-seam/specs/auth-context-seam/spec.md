## Purpose

Defines the platform-owned request-context seam: modules resolve the authenticated identity, organization, and account IDs for the current request through `internal/platform/authcontext`, without importing the auth module.

## ADDED Requirements

### Requirement: Request context read through platform seam

The system SHALL expose request identity and organization context accessors in `internal/platform/authcontext`. Modules other than `auth` SHALL read the resolved request context (`OrganizationID`, `AccountID`, identity) via that package rather than importing `internal/modules/auth` for context reads.

#### Scenario: Handler resolves organization ID without auth import

- **WHEN** a handler needs the current organization ID for a request
- **THEN** it SHALL use `authcontext.GetRequestContext` or `authcontext.GetOrganizationID`
- **AND** the file SHALL NOT depend on `internal/modules/auth` solely for context access

#### Scenario: Auth middleware populates the seam

- **WHEN** the auth middleware authenticates a request and resolves database IDs
- **THEN** it SHALL store the identity and request context via the `authcontext` package
- **AND** handlers SHALL read the same values through `authcontext` accessors

#### Scenario: Context propagates through service layers

- **WHEN** request context or identity must reach a service layer
- **THEN** it SHALL be propagated via the `authcontext.WithRequestContext` / `WithIdentity` context helpers
- **AND** read back via `RequestContextFromContext` / `IdentityFromContext`
