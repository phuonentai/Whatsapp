## ADDED Requirements


### Requirement: Cross-organization data isolation is E2E-tested

The E2E tests SHALL verify that data created by one seeded organization is invisible to another seeded organization at the API level.

#### Scenario: Org A data absent from Org B list

- **WHEN** an org creates a contact under its own seeded org
- **THEN** the same contact SHALL NOT appear in another org's contacts list

### Requirement: Pagination behavior is E2E-tested

The E2E tests SHALL verify the CRM list pagination contract: default `limit` of 20, explicit `limit`/`offset` parameters, and full result retrieval beyond the default page size.

#### Scenario: Default limit returns 20 rows

- **WHEN** an org has more than 20 contacts and a list request is made without `limit`/`offset`
- **THEN** exactly 20 rows SHALL be returned

#### Scenario: Explicit limit and offset retrieve the remainder

- **WHEN** a list request specifies `limit` and `offset` beyond the first page
- **THEN** the remaining rows SHALL be returned

### Requirement: Outbound reply persistence is E2E-tested

The E2E tests SHALL verify that sending a reply via `POST /crm/conversaciones/:id/mensajes` persists an outbound message retrievable through the messages API.

#### Scenario: Reply persists as an outbound message

- **WHEN** a reply is sent to an existing conversation via the messages endpoint
- **THEN** the persisted message retrieved via `/crm/conversaciones/:id/mensajes` SHALL have `direction` equal to `outbound`

### Requirement: Mock-auth guard is E2E-tested

The E2E tests SHALL verify that, with `AUTH_MOCK_ENABLED`, a request without an `X-Test-Org-ID` header is rejected with 401.

#### Scenario: Missing mock header returns 401

- **WHEN** a request is made with `AUTH_MOCK_ENABLED=true` and no `X-Test-Org-ID` header
- **THEN** the response SHALL have status 401

### Requirement: RBAC boundary is E2E-tested

The E2E tests SHALL verify that a member without the `org:manage` permission cannot access org-management-gated endpoints, returning 403.

#### Scenario: Member access to org-manage-gated endpoint is rejected with 403

- **WHEN** a member account attempts to access an endpoint gated by the `org:manage` permission (e.g., `GET /api/v1/whatsapp/config`)
- **THEN** the response SHALL have status 403
