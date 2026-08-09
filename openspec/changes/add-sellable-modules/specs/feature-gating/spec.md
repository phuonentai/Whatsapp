## MODIFIED Requirements

### Requirement: FeatureProvider derives enabled features from subscription plan and purchased modules

The system SHALL derive the organization's enabled feature set through `FeatureProvider.GetEntitlement(ctx, orgID)`, which returns an `Entitlement` containing `Features` (plan features unioned with features granted by purchased modules), `Quotas`, `Usage`, `IsReadOnly`, `IsGracePeriod`, and `PlanName`. Features are derived per request and cached once per request lifecycle. Module-granted feature keys SHALL be namespaced (e.g., `tickets_module`) and SHALL NOT collide with plan feature keys.

#### Scenario: Free tier gets no CRM features

- **WHEN** an organization has no active subscription (or free plan)
- **THEN** all CRM feature flags SHALL be false
- **AND** all module feature flags SHALL be false
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

#### Scenario: Purchased module adds features to any plan

- **WHEN** an organization with an active subscription has the `tickets` module enabled
- **THEN** the `tickets_module` feature flag SHALL be true
- **AND** plan-derived features SHALL remain unchanged by module enablement

#### Scenario: No active subscription disables module features

- **WHEN** an organization has no active subscription
- **THEN** module feature flags SHALL be false regardless of module metadata present

### Requirement: Plan and module feature mapping is centralized and configurable

The system SHALL keep plan-to-feature and module-to-feature mappings in centralized, data-driven sources: subscription plan metadata (the existing `crm_features` metadata pattern) for plan features, and the module registry for module features. Unknown plan names SHALL default to the free tier feature set.

#### Scenario: Adding a new plan updates feature availability

- **WHEN** a new plan name with `crm_features` metadata is provisioned in the billing provider
- **THEN** organizations on that plan SHALL receive the mapped features without code changes elsewhere

#### Scenario: Unknown plan name defaults to free tier

- **WHEN** a subscription has a plan name not recognized by the billing provider
- **THEN** the system SHALL default to the free tier feature set (no features)

#### Scenario: Module metadata parsed independently of plan features

- **WHEN** subscription metadata contains both `crm_features` and a module key (e.g., `module_tickets`)
- **THEN** plan features SHALL be parsed from `crm_features` and module features SHALL be parsed from the module key
- **AND** both SHALL be present in the resulting entitlement

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

The system SHALL store the entitlement in the Gin context after middleware execution, making it available to downstream handlers via `features.GetEntitlement(c)`.

#### Scenario: Handler reads feature state from context

- **WHEN** a handler calls `features.GetEntitlement(c)`
- **THEN** the system SHALL return the `*Entitlement` for the organization, including features, quotas, usage, and module state

### Requirement: Entitlement endpoint returns the enabled feature set

The system SHALL provide `GET /api/crm/entitlement` returning the enabled features, quotas, usage, subscription state, and module state for the authenticated organization.

#### Scenario: Pro tier entitlement response

- **WHEN** a Pro-tier organization without modules requests `GET /api/crm/entitlement`
- **THEN** the response SHALL include `crm_contacts_manage`, `crm_companies`, `crm_deals`, `crm_activities` as true and `crm_tags` as false
- **AND** the response SHALL include no enabled modules

#### Scenario: Free tier entitlement response

- **WHEN** a free organization requests `GET /api/crm/entitlement`
- **THEN** the response SHALL include all CRM feature flags as false
- **AND** the response SHALL include no enabled modules

### Requirement: FeatureService caches features per request

The system SHALL derive and cache the entitlement once per request via middleware, avoiding repeated subscription reads within a single request lifecycle.

#### Scenario: Multiple handler checks within one request read features once

- **WHEN** a request triggers multiple `features.Require` / module-gating checks for the same org
- **THEN** the subscription and module state SHALL be read at most once per request

### Requirement: WhatsApp Activity creation respects feature flag

The system SHALL NOT create Activity records for inbound WhatsApp messages when `crm_activities` is disabled for the organization.

#### Scenario: Activities disabled — no Activity created on WhatsApp message

- **WHEN** a WhatsApp message arrives for an organization with `crm_activities` disabled
- **THEN** the Contact, Conversation, and Message SHALL still be persisted
- **AND** no Activity record SHALL be created

### Requirement: Feature flags are independent of RBAC permissions

The system SHALL enforce feature flags separately from RBAC permissions. Feature flags control what is available per subscription tier and module purchases; permissions control what a user can do within available features.

#### Scenario: Admin on Starter plan cannot access deals

- **WHEN** an admin user on a Starter plan (no `crm_deals`) attempts to access `/api/crm/deals`
- **THEN** the feature middleware SHALL return 403 before the permission middleware runs
- **AND** the `deal:manage` permission is irrelevant (feature is unavailable)

#### Scenario: Member on Pro plan with deal:view but not deal:manage

- **WHEN** a member user on a Pro plan (`crm_deals` enabled) with `deal:view` but not `deal:manage` attempts to create a deal
- **THEN** the feature middleware SHALL pass
- **AND** the permission middleware SHALL return 403 (insufficient permission)
