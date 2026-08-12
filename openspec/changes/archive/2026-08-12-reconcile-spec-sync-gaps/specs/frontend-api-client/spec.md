## ADDED Requirements

### Requirement: API client supports query parameters
The `RequestOptions` type SHALL accept a `params` record of string/number/undefined values. The client SHALL serialize non-undefined entries into the request URL query string using `URLSearchParams` encoding, joining with `?` when the endpoint has no query string and `&` when it already has one.

#### Scenario: Query params appended to bare endpoint

- **WHEN** a repository calls `apiClient.get("/crm/contactos", { params: { lead_status: "cliente", limit: 50 } })`
- **THEN** the request URL SHALL be `/crm/contactos?lead_status=cliente&limit=50`

#### Scenario: Undefined params skipped

- **WHEN** a repository calls `apiClient.get("/crm/contactos", { params: { source: undefined, limit: 10 } })`
- **THEN** the request URL SHALL NOT contain `source`
- **AND** the URL SHALL be `/crm/contactos?limit=10`

#### Scenario: Params joined to existing query string

- **WHEN** a repository calls `apiClient.get("/crm/contactos/search", { params: { q: "abc" } })` against an endpoint that already carries a query string
- **THEN** the params SHALL be appended with `&` after the existing query string

### Requirement: Query key namespaces match hook usage
The `queryKeys` object SHALL expose a `whatsappConfig` namespace with `all` and `detail()` keys matching the hook call sites (`use-whatsapp-config-query.ts`, `use-toggle-whatsapp-config.ts`, `use-upsert-whatsapp-config.ts`). Existing namespaces SHALL remain unchanged.

#### Scenario: whatsappConfig keys resolve

- **WHEN** a hook references `queryKeys.whatsappConfig.all` or `queryKeys.whatsappConfig.detail()`
- **THEN** both SHALL exist and type-check under strict TypeScript

