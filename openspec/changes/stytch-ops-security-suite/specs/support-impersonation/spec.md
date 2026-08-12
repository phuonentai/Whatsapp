## ADDED Requirements

### Requirement: Support member can impersonate an org member

A member with `org:manage` plus the `support:impersonate` permission SHALL be able to start an impersonation session for a member of the same organization. The impersonation token SHALL be minted outside the application (Stytch dashboard support console, or the verified API variant), exchanged by the Go backend via `POST /v1/b2b/impersonation/authenticate` behind the existing circuit breaker, and the returned session JWT SHALL be set with the standard session cookies. The impersonated session SHALL be limited to 60 minutes and SHALL NOT be extendable.

#### Scenario: Support member impersonates a member

- **WHEN** a support member with `org:manage` + `support:impersonate` starts an impersonation for a member of the same org
- **THEN** the backend SHALL exchange the impersonation token via `POST /v1/b2b/impersonation/authenticate`
- **AND** the impersonated session SHALL be established with the standard session cookies

#### Scenario: Impersonation is org-scoped

- **WHEN** an impersonation token targets a member in a different organization than the impersonator
- **THEN** the backend SHALL reject the impersonation
- **AND** the impersonator's session SHALL be unaffected

#### Scenario: Breaker open during impersonation exchange

- **WHEN** the Stytch circuit breaker is open during the impersonation exchange
- **THEN** the backend SHALL return a 503 structured error (`impersonation_unavailable`)
- **AND** no impersonated session SHALL be established

#### Scenario: Impersonation session is non-extendable

- **WHEN** an impersonated session approaches its 60-minute lifetime
- **THEN** it SHALL expire per Stytch enforcement
- **AND** the application SHALL NOT extend or refresh it

### Requirement: Impersonated sessions are visibly flagged

Impersonated session JWTs SHALL be detected by the auth middleware via their impersonation claims (`impersonating`, `impersonator_id`, `impersonator_email_address`). While an impersonated session is active, the UI SHALL render a persistent "viewing as <member>" banner with an exit action that revokes the session and returns to the impersonator's own session.

#### Scenario: UI shows viewing-as banner

- **WHEN** a request carries an impersonated session JWT
- **THEN** the UI SHALL render the "viewing as <member>" banner on all protected pages
- **AND** the banner SHALL offer an exit action

#### Scenario: Exit revokes the impersonated session

- **WHEN** the support member clicks exit on the banner
- **THEN** the backend SHALL revoke the impersonated session
- **AND** the support member SHALL be returned to their own authenticated session

### Requirement: Impersonation is audited

The system SHALL record `support_impersonation_started` and `support_impersonation_ended` audit rows (`tipo=sistema`) attributed to the impersonated member's organization, with bounded detail including the impersonator's `stytch_member_id`. Audit recording SHALL be best-effort and non-blocking, per the `admin-panel-audit-log` contract.

#### Scenario: Start and end recorded

- **WHEN** a support member starts and later exits an impersonation
- **THEN** the audit stream SHALL contain `support_impersonation_started` and `support_impersonation_ended` rows
- **AND** each row SHALL carry the impersonator's `stytch_member_id` and the target member's `stytch_member_id`

#### Scenario: Audit failure does not block impersonation

- **WHEN** the audit write fails during impersonation
- **THEN** the impersonation SHALL still proceed
- **AND** the failure SHALL be logged as a warning

### Requirement: No local credential storage

Impersonation tokens, impersonated sessions, and impersonator credentials SHALL NOT be persisted locally. Audit rows SHALL contain only member IDs, organization ID, event type, timestamp, and bounded non-sensitive detail.

#### Scenario: No token or session material in storage

- **WHEN** an impersonation occurs
- **THEN** no local database row SHALL store the impersonation token, session token, or session JWT
