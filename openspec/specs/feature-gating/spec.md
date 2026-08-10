## Purpose

Defines subscription-plan feature gating: FeatureService derivation, centralized plan-to-feature mapping, and API route middleware.

## Requirements

### Requirement: FeatureService derives enabled features from subscription plan

The system SHALL provide a `FeatureService` at the platform level that derives a set of boolean feature flags from the organization's active subscription plan.

#### Scenario: Free tier gets no CRM features

- **WHEN** an organization has no active subscription (or free plan)
- **THEN** `FeatureService.IsEnabled(orgID, "crm_contacts_manage")` SHALL return false
- **AND** all CRM feature flags SHALL be false
- **AND** the WhatsApp bridge (`crm_contacts` entity) SHALL still function (it is not a feature flag)

#### Scenario: Starter plan gets basic contact management

- **WHEN** an organization has an active Starter subscription
- **THEN** `crm_contacts_manage` SHALL be true
- **AND** `crm_companies`, `crm_deals`, `crm_activities`, `crm_tags` SHALL be false

#### Scenario: Pro plan gets full CRM except tags

- **WHEN** an organization has an active Pro subscription
- **THEN** `crm_contacts_manage`, `crm_companies`, `crm_deals`, `crm_activities` SHALL be true
- **AND** `crm_tags` SHALL be false

#### Scenario: Enterprise plan gets all features

- **WHEN** an organization has an active Enterprise subscription
- **THEN** all CRM feature flags SHALL be true

### Requirement: Plan-to-feature mapping is centralized and configurable

The system SHALL define the mapping between subscription plan names and CRM feature sets in a single source-of-truth file (`internal/platform/features/plans.go`).

#### Scenario: Adding a new plan updates feature availability

- **WHEN** a new plan name is added to the plan-to-feature mapping
- **THEN** organizations on that plan SHALL receive the mapped features without code changes elsewhere

#### Scenario: Unknown plan name defaults to free tier

- **WHEN** a subscription has a plan name not in the mapping
- **THEN** the system SHALL default to the free tier feature set (no features)

### Requirement: Feature middleware gates API routes

The system SHALL provide `features.Require(featureName)` middleware that returns HTTP 403 if the feature is not enabled for the requesting organization.

#### Scenario: Feature enabled allows access

- **WHEN** a request passes through `features.Require("crm_deals")` and the organization has `crm_deals` enabled
- **THEN** the request SHALL proceed to the handler

#### Scenario: Feature disabled returns 403

- **WHEN** a request passes through `features.Require("crm_deals")` and the organization does not have `crm_deals` enabled
- **THEN** the system SHALL return HTTP 403 with a JSON body `{"error": "feature_disabled", "feature": "crm_deals"}`
- **AND** SHALL abort the request (handler not called)

### Requirement: Feature context is available to handlers

The system SHALL store enabled features in the Gin context after middleware execution, making them available to downstream handlers via `features.GetFeatures(c)`.

#### Scenario: Handler reads feature state from context

- **WHEN** a handler calls `features.GetFeatures(c)`
- **THEN** the system SHALL return a `map[string]bool` of all feature flags for the organization

### Requirement: Feature endpoint returns enabled feature set

The system SHALL provide `GET /api/crm/features` returning the set of enabled CRM features for the authenticated organization.

#### Scenario: Pro tier features response

- **WHEN** a Pro-tier organization requests `GET /api/crm/features`
- **THEN** the response SHALL be `{"crm_contacts_manage": true, "crm_companies": true, "crm_deals": true, "crm_activities": true, "crm_tags": false}`

#### Scenario: Free tier features response

- **WHEN** a free organization requests `GET /api/crm/features`
- **THEN** the response SHALL be `{"crm_contacts_manage": false, "crm_companies": false, "crm_deals": false, "crm_activities": false, "crm_tags": false}`

### Requirement: FeatureService caches features per request

The system SHALL derive and cache the feature set once per request via middleware, avoiding repeated subscription reads within a single request lifecycle.

#### Scenario: Multiple handler checks within one request read features once

- **WHEN** a request triggers multiple `FeatureService.IsEnabled()` calls for the same org
- **THEN** the subscription SHALL be read at most once per request

### Requirement: WhatsApp Activity creation respects feature flag

The system SHALL NOT create Activity records for inbound WhatsApp messages when `crm_activities` is disabled for the organization.

#### Scenario: Activities disabled — no Activity created on WhatsApp message

- **WHEN** a WhatsApp message arrives for an organization with `crm_activities` disabled
- **THEN** the Contact, Conversation, and Message SHALL still be persisted
- **AND** no Activity record SHALL be created

### Requirement: Feature flags are independent of RBAC permissions

The system SHALL enforce feature flags separately from RBAC permissions. Feature flags control what is available per subscription tier; permissions control what a user can do within available features.

#### Scenario: Admin on Starter plan cannot access deals

- **WHEN** an admin user on a Starter plan (no `crm_deals`) attempts to access `/api/crm/deals`
- **THEN** the feature middleware SHALL return 403 before the permission middleware runs
- **AND** the `deal:manage` permission is irrelevant (feature is unavailable)

#### Scenario: Member on Pro plan with deal:view but not deal:manage

- **WHEN** a member user on a Pro plan (`crm_deals` enabled) with `deal:view` but not `deal:manage` attempts to create a deal
- **THEN** the feature middleware SHALL pass
- **AND** the permission middleware SHALL return 403 (insufficient permission)

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
