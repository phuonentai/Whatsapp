## ADDED Requirements

### Requirement: Entitlement exposes AI usage and credit state

The system SHALL include AI token consumption and credit allowance in the `Entitlement` returned by `FeatureProvider.GetEntitlement`, so AI usage is visible in the same tenant context as feature flags. Feature flags and AI metering remain separate concerns: flags control availability, the ledger controls consumption.

#### Scenario: AI usage present in entitlement usage map

- **WHEN** `FeatureProvider.GetEntitlement` is called for an organization that has consumed AI tokens
- **THEN** the returned `Entitlement.Usage` SHALL contain `ai_tokens_input`, `ai_tokens_output`, `ai_tokens_embedding`, `ai_credits_used`, and `ai_credits_remaining` reflecting the current billing period ledger state

#### Scenario: AI credit allowance present in entitlement quotas map

- **WHEN** an organization's subscription metadata defines `ai_credits_max`
- **THEN** the returned `Entitlement.Quotas` SHALL contain `ai_credits` equal to that allowance

#### Scenario: Zero usage for untouched organizations

- **WHEN** `GetEntitlement` is called for an organization with no AI consumption
- **THEN** the AI usage entries SHALL be present and zero

### Requirement: AI feature flag gates AI routes

The system SHALL gate AI-facing routes behind an `ai_assistant` feature flag derived from subscription metadata, enforced before the credit guard runs.

#### Scenario: Flag enabled allows access

- **WHEN** an organization has `ai_assistant` enabled in its subscription metadata
- **THEN** requests to guarded AI routes SHALL pass the feature middleware

#### Scenario: Flag disabled returns 403

- **WHEN** an organization does not have `ai_assistant` enabled
- **THEN** requests to guarded AI routes SHALL return HTTP 403 from the platform feature middleware (`features.Require`), whose JSON body identifies the feature as unavailable (error `funcionalidad_no_disponible`, field `funcionalidad` = `ai_assistant`)
- **AND** the credit guard SHALL NOT run (feature check precedes usage enforcement)
