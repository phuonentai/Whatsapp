## ADDED Requirements

### Requirement: API client supports PATCH requests

The frontend `ApiClient` SHALL provide a `patch<T>(endpoint, body?, options?)` method equivalent in shape to the existing `put` but issuing HTTP `PATCH`. The request body SHALL be optional (PATCH requests without a body, such as the WhatsApp config toggle, SHALL be supported).

#### Scenario: PATCH with body

- **WHEN** a repository calls `apiClient.patch("/crm/conversaciones/1/status", { status: "closed" })`
- **THEN** the client SHALL send an HTTP PATCH to that endpoint with the JSON body
- **AND** the response SHALL be typed as `T`

#### Scenario: PATCH without body

- **WHEN** a repository calls `apiClient.patch("/whatsapp/config/toggle")`
- **THEN** the client SHALL send an HTTP PATCH with no body

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

### Requirement: Frontend build is green

The frontend SHALL compile with zero TypeScript errors under strict mode, and `pnpm lint` / `pnpm build` SHALL pass.

#### Scenario: Typecheck passes

- **WHEN** `npx tsc --noEmit -p tsconfig.json` runs
- **THEN** it SHALL exit with zero errors

#### Scenario: Build and lint pass

- **WHEN** `pnpm lint` and `pnpm build` run
- **THEN** both SHALL complete successfully
