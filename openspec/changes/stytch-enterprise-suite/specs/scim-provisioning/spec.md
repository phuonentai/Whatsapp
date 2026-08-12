## ADDED Requirements

### Requirement: SCIM connections managed via Admin Portal

Settings SHALL expose a SCIM provisioning view (`?view=scim`) rendering the Stytch Admin Portal SCIM component (`AdminPortalSCIMMountOptions` from `@stytch/vanilla-js/b2b/adminPortal`), gated by `org:manage`. Organization admins SHALL create, view, and delete SCIM connections through the Admin Portal, including the per-connection SCIM base URL and bearer token. SCIM bearer tokens SHALL be shown only by Stytch, SHALL NOT be transmitted to or stored by the platform's servers, and SHALL be handled entirely within the Admin Portal surface.

#### Scenario: Admin opens the SCIM view

- **WHEN** a member with `org:manage` navigates to `/settings?view=scim`
- **THEN** the page SHALL render the Admin Portal SCIM management component
- **AND** the connection list SHALL reflect the org's SCIM connections

#### Scenario: Admin creates a SCIM connection

- **WHEN** an admin completes the SCIM connection form in the Admin Portal
- **THEN** Stytch SHALL provision the connection with a SCIM base URL and bearer token
- **AND** the connection SHALL appear in the org's SCIM connection list

#### Scenario: SCIM bearer token never stored locally

- **WHEN** a SCIM connection is created or its token is shown
- **THEN** the bearer token SHALL be displayed only by the Admin Portal
- **AND** no local database row SHALL store the SCIM bearer token

#### Scenario: Member without org:manage cannot manage SCIM

- **WHEN** a member without `org:manage` attempts to open `/settings?view=scim`
- **THEN** the view SHALL NOT be reachable and its settings entry SHALL NOT be rendered

### Requirement: SCIM group-to-role mapping

SCIM group-to-role mapping SHALL be configurable per connection through the Admin Portal, mapping IdP groups to Stytch roles defined in the RBAC policy. Members provisioned via SCIM SHALL receive the role mapped to their groups; members without a mapped group SHALL default to `stytch_member`.

#### Scenario: IdP group maps to a Stytch role

- **WHEN** an admin maps an IdP group to a Stytch role in the Admin Portal
- **AND** a member is provisioned belonging to that group
- **THEN** the member SHALL be assigned the mapped role

#### Scenario: Provisioned member without group mapping

- **WHEN** a member is SCIM-provisioned with no group mapping
- **THEN** the member SHALL hold the default `stytch_member` role

### Requirement: SCIM lifecycle events reflected in member list

Members provisioned, updated, or deprovisioned via SCIM SHALL appear correctly in the platform's member-facing surfaces: provisioned members SHALL be joinable/sign-in capable per the org's auth settings, updated members SHALL reflect role changes, and deprovisioned members SHALL lose access consistent with Stytch's enforcement at authentication time.

#### Scenario: SCIM deprovisioned member loses access

- **WHEN** a member is deprovisioned via SCIM
- **THEN** subsequent authentication attempts SHALL fail per Stytch's enforcement
- **AND** the platform SHALL NOT grant access on stale local state
