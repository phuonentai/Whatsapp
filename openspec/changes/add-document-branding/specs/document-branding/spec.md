## ADDED Requirements

### Requirement: Org-scoped document branding configuration

The system SHALL store a single branding configuration per organization in `document_branding.org_branding` with: logo file asset reference, primary/accent colors, letterhead text, terms footer text, default validity days, default IVA percentage, and document numbering prefix. Branding SHALL be readable with `org:view` and writable with `org:manage`; writes SHALL emit a `branding_updated` audit event with the acting `stytch_member_id`.

#### Scenario: Member reads org branding
- **WHEN** a member with `org:view` calls `GET /api/branding`
- **THEN** the response SHALL contain the organization's branding configuration

#### Scenario: Member updates branding
- **WHEN** an `org:manage` member updates colors/letterhead/terms via `PUT /api/branding`
- **THEN** the configuration SHALL be persisted with `updated_at` bumped
- **AND** a `branding_updated` audit event SHALL record the acting member and changed fields

#### Scenario: Branding write denied without org:manage
- **WHEN** a member without `org:manage` attempts to update branding
- **THEN** the system SHALL return HTTP 403 with a Spanish error message
- **AND** SHALL NOT modify the configuration

### Requirement: Logo upload with validation

The system SHALL accept a logo upload via `POST /api/branding/logo` (multipart) for `org:manage` members, validate content type (PNG/JPEG only; SVG rejected), file size (≤ 2MB), and dimension sanity, store the bytes via the file-asset manager with a branding category, and reference the resulting asset from the org's branding row. Replacing the logo SHALL emit a `logo_updated` audit event.

#### Scenario: Valid logo uploaded
- **WHEN** an `org:manage` member uploads a PNG or JPEG logo within size/dimension limits
- **THEN** the file SHALL be stored via the file-asset manager
- **AND** the branding row SHALL reference the new asset
- **AND** a `logo_updated` audit event SHALL be recorded

#### Scenario: Invalid logo type rejected
- **WHEN** a member uploads an SVG or other unsupported type
- **THEN** the system SHALL return HTTP 400 with a Spanish error
- **AND** SHALL NOT store the file or modify the branding row

### Requirement: Branding exposed through a domain provider interface

The system SHALL expose branding to document renderers through a `DocumentBrandingProvider` domain interface (`GetBranding(ctx, orgID)` returning the resolved value object, logo URL already resolved by the infra layer). Domain models of other capabilities SHALL NOT import file assets or storage; renderers SHALL consume branding exclusively through this interface.

#### Scenario: Renderer resolves org branding
- **WHEN** a document renderer calls the provider for an organization with branding configured
- **THEN** the provider SHALL return the resolved branding value object including the logo URL

#### Scenario: Organization without branding gets defaults
- **WHEN** a renderer requests branding for an organization that never configured it
- **THEN** the provider SHALL return a default configuration (no error), with the org's company name/NIT from existing company data

### Requirement: Branding time-snapshot key

The system SHALL expose the branding `updated_at` as a snapshot key (`org_id:updated_at`) so document capabilities can capture branding at issue time. Documents SHALL reference the snapshot key of the branding active when issued.

#### Scenario: Document captures branding snapshot
- **WHEN** a document (e.g., a quote) is created with branding active
- **THEN** the document SHALL store the branding snapshot key
- **AND** later branding changes SHALL NOT affect the rendering of that document unless explicitly re-issued
