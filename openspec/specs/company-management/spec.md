## Purpose

Defines the company entity with organization scoping, Colombian-specific fields, and an owner assignment.

## Requirements

### Requirement: Company entity with organization scoping

The system SHALL store CRM companies in `crm.companies` scoped by `organization_id` with a unique constraint on `(organization_id, name)`.

#### Scenario: Company is created for an organization

- **WHEN** a company is created with name, NIT, sector, and tipo_empresa for an organization
- **THEN** the company SHALL be persisted with all fields
- **AND** the company SHALL be scoped to the requesting organization

#### Scenario: Duplicate company name in same organization is rejected

- **WHEN** a company is created with a name that already exists for the organization
- **THEN** the system SHALL return a conflict error: "Ya existe una empresa con este nombre."

#### Scenario: Same company name in different organizations is allowed

- **WHEN** two different organizations each create a company with the same name
- **THEN** both companies SHALL be created successfully

### Requirement: Company has Colombian-specific fields

The system SHALL store: `nit` (VARCHAR, Número de Identificación Tributaria with dígito de verificación), `tipo_empresa` (VARCHAR, CHECK: microempresa/pequeña/mediana/grande), `sector` (VARCHAR), `ciudad` (VARCHAR), `departamento` (VARCHAR), in addition to `name` (required), `website`, `notes`, and `metadata` (JSONB).

#### Scenario: Company created with NIT and Colombian classification

- **WHEN** a company is created with `nit = "900.123.456-7"`, `tipo_empresa = "mediana"`, `sector = "Tecnología"`, `ciudad = "Medellín"`, `departamento = "Antioquia"`
- **THEN** all fields SHALL be persisted

#### Scenario: NIT is optional

- **WHEN** a company is created without a NIT (e.g., informal business or foreign company)
- **THEN** the company SHALL be created successfully with `nit = NULL`

#### Scenario: Invalid tipo_empresa value is rejected

- **WHEN** a company is created with `tipo_empresa = "corporación"`
- **THEN** the system SHALL return a validation error: "Tipo de empresa inválido. Valores permitidos: microempresa, pequeña, mediana, grande."

### Requirement: Company has an owner

The system SHALL allow a company to have an `owner_account_id` referencing `organizations.accounts(id)`.

#### Scenario: Company assigned to an account (responsable)

- **WHEN** a company is created or updated with `owner_account_id` set
- **THEN** the company SHALL be associated with that account as its responsable

### Requirement: Company list supports pagination and search

The system SHALL provide a paginated company list endpoint with optional search by name, NIT, sector, or ciudad.

#### Scenario: Companies searched by NIT

- **WHEN** a GET request is made to `/api/crm/companies?search=900.123`
- **THEN** the system SHALL return companies whose name, NIT, or sector contains "900.123" (case-insensitive)

### Requirement: Company detail includes contact and deal counts

The system SHALL include `total_contactos` and `total_negocios` when returning a company detail.

#### Scenario: Company detail with counts

- **WHEN** a GET request is made to `/api/crm/companies/1`
- **THEN** the response SHALL include `total_contactos` and `total_negocios` fields

### Requirement: Company CRUD is RBAC-protected

The system SHALL require `contact:view` permission for read operations and `contact:manage` permission for write operations on companies.

#### Scenario: User without manage permission sees error in Spanish

- **WHEN** a user without `contact:manage` permission attempts to create a company
- **THEN** the system SHALL return HTTP 403 with message "No tienes permiso para gestionar empresas."
