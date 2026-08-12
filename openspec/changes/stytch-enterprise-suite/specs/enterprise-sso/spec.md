## ADDED Requirements

### Requirement: SSO connections managed via Admin Portal

Settings SHALL expose an SSO management view (`?view=sso`) rendering the Stytch Admin Portal SSO component (`AdminPortalSSOMountOptions` from `@stytch/vanilla-js/b2b/adminPortal`), gated by `org:manage`. Organization admins SHALL create, edit, and delete SAML and OIDC connections through the Admin Portal. Connection secrets (certificates, client secrets, issuer URLs) SHALL live only in Stytch and SHALL NOT be transmitted to or stored by the platform's servers.

#### Scenario: Admin opens the SSO view

- **WHEN** a member with `org:manage` navigates to `/settings?view=sso`
- **THEN** the page SHALL render the Admin Portal SSO management component
- **AND** the connection list SHALL reflect the org's Stytch SSO connections

#### Scenario: Admin creates a SAML connection

- **WHEN** an admin completes the SAML connection form in the Admin Portal
- **THEN** Stytch SHALL create the connection via the SSO API
- **AND** the connection SHALL appear in the org's SSO connection list

#### Scenario: Member without org:manage cannot manage SSO

- **WHEN** a member without `org:manage` attempts to open `/settings?view=sso`
- **THEN** the view SHALL NOT be reachable and its settings entry SHALL NOT be rendered

#### Scenario: SSO secrets never stored locally

- **WHEN** any SSO connection is created or edited
- **THEN** certificates, client secrets, and issuer metadata SHALL be handled exclusively by Stytch
- **AND** no local database row SHALL store SSO credential material

#### Scenario: Admin Portal availability is verified before adoption

- **WHEN** the SSO view is implemented
- **THEN** the availability of `AdminPortalSSOMountOptions`/`client.portal` in the pinned `@stytch/vanilla-js` SHALL be verified and the outcome recorded (task 3.2)
- **AND** if the exports are unavailable in a reviewed version, the documented fallback (Go-backed SSO connection CRUD) SHALL be implemented instead and this capability's deltas SHALL be revised to match

### Requirement: SSO JIT provisioning per org

An organization's SSO JIT provisioning SHALL be enabled by the org admin through the governed auth-policy surface (the settings `?view=access` JIT card, shown when the org has at least one active SSO connection), writing `sso_jit_provisioning: RESTRICTED` scoped to `sso_jit_provisioning_allowed_connections` (the org's active connection ids) via `PUT /v1/b2b/organizations/{organization_id}`. The default SHALL remain `NOT_ALLOWED` until the admin opts in. Members JIT-provisioned via the IdP SHALL receive Stytch's default `stytch_member` role (least privilege); the Admin Portal connection-creation surface SHALL NOT auto-write org policy (it runs client-side with no backend hook). Org-wide `ALL_ALLOWED` SSO JIT SHALL NOT be used.

#### Scenario: Employee authenticates via the IdP

- **WHEN** an employee of an SSO-enabled org authenticates through one of the org's allowed connections
- **THEN** Stytch SHALL JIT-provision the member into the org with the default `stytch_member` role
- **AND** the member SHALL be able to complete authentication without a pre-existing invite

#### Scenario: Admin enables SSO JIT

- **WHEN** an admin with `org:manage` enables SSO JIT for an org that has active SSO connections and saves
- **THEN** the org SHALL be updated with `sso_jit_provisioning: RESTRICTED` and `sso_jit_provisioning_allowed_connections` set to the org's active connection ids
- **AND** the settings UI SHALL reflect the saved policy (display-only mirror)

#### Scenario: SSO JIT default is disabled and least privilege

- **WHEN** an org has never enabled SSO JIT
- **THEN** its `sso_jit_provisioning` SHALL remain `NOT_ALLOWED`
- **AND** a member SHALL NOT be provisioned by SSO without a pre-existing invite or membership

#### Scenario: SSO JIT default role is least privilege

- **WHEN** a member is SSO JIT-provisioned
- **THEN** the member SHALL hold Stytch's default implicit `stytch_member` role and nothing beyond it
- **AND** role escalation SHALL remain admin-managed via the member API

#### Scenario: Auth-policy write is breaker-guarded

- **WHEN** the Stytch circuit breaker is open during an SSO-JIT policy update
- **THEN** the update SHALL be rejected with a 503 structured error (`auth_policy_update_unavailable`)
- **AND** the local display mirror SHALL NOT be modified

### Requirement: SSO as org primary auth method

Organizations SHALL be able to configure SSO as a primary authentication method via the auth-policy settings surface, writing `auth_methods: RESTRICTED` together with `allowed_auth_methods` including `sso` (and optionally `sso_default_connection_id`). Members of orgs where SSO is primary SHALL authenticate through the IdP; the fallback methods configured for the org remain available. The first policy write SHALL preserve the org's current effective method set while adding the requested methods.

#### Scenario: Org sets SSO primary

- **WHEN** an admin configures the org's allowed primary methods to include `sso` and saves
- **THEN** the org SHALL be written with `auth_methods: RESTRICTED` and `allowed_auth_methods` including `sso`
- **AND** the Stytch login surface SHALL present the IdP as a login option

#### Scenario: First write preserves existing methods

- **WHEN** an org whose `auth_methods` is currently `ALL_ALLOWED` performs its first policy write
- **THEN** the persisted `allowed_auth_methods` SHALL include the org's current effective method set in addition to the requested changes

#### Scenario: Auth-policy write is breaker-guarded

- **WHEN** the Stytch circuit breaker is open during an auth-method update
- **THEN** the update SHALL be rejected with a 503 structured error (`auth_policy_update_unavailable`)
- **AND** the local display mirror SHALL NOT be modified
