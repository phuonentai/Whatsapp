# Spec: ai-onboarding

## Purpose

TBD. Defines the guided AI-first signup wizard, the first-run dashboard checklist, and the assistant introduction moment that onboards new organizations into the WhatsApp copilot.

## Requirements

### Requirement: Guided AI-first signup wizard

The signup flow SHALL guide the user through account, organization, and business-context steps, then submit through the existing Stytch-compliant bootstrap endpoint. The flow SHALL reuse the existing signup contract (native Stytch invite via `Members.Create` with `SendInvite: true`, no `owner_password`, structured error codes) and SHALL NOT introduce any password or local credential capture.

#### Scenario: User completes all wizard steps

- **WHEN** a new user completes the account, organization, and business-context steps
- **THEN** the flow SHALL submit the existing signup request with owner, organization, and industry fields
- **AND** SHALL NOT include an `owner_password` field
- **AND** SHALL route verification through the Stytch native invite magic link

#### Scenario: Business context is captured client-side

- **WHEN** a user answers the WhatsApp-readiness and business-goal prompts in the wizard
- **THEN** the answers SHALL be stored client-side (localStorage) and SHALL be available to the first-run checklist
- **AND** SHALL NOT be transmitted in the signup payload or persisted server-side

### Requirement: First-run onboarding checklist

The system SHALL render a first-run checklist on the dashboard for new organizations, with steps to connect WhatsApp, choose a plan, meet the AI assistant, and open the inbox. Each step SHALL reflect real completion state derived from existing data, and SHALL disappear once all steps are complete.

#### Scenario: Checklist appears for incomplete first run

- **WHEN** an organization has not completed all first-run steps
- **THEN** the dashboard SHALL render the checklist with per-step status (done/todo)
- **AND** each step SHALL link to the relevant surface (Settings → WhatsApp, plans modal, assistant intro, inbox)

#### Scenario: WhatsApp step completes when connected

- **WHEN** an active WhatsApp configuration exists for the organization
- **THEN** the connect-WhatsApp checklist step SHALL be marked done

#### Scenario: Plan step completes when subscribed

- **WHEN** the organization has an active subscription
- **THEN** the choose-a-plan checklist step SHALL be marked done

#### Scenario: Checklist hides when complete

- **WHEN** all first-run steps are complete
- **THEN** the checklist SHALL no longer render

### Requirement: AI assistant introduction

The system SHALL present a dismissible "Meet your assistant" moment for new organizations that explains the WhatsApp copilot (which drafts replies for a human to approve) and the knowledge base, and links to the agent settings.

#### Scenario: Assistant intro shown for new orgs

- **WHEN** a new organization opens the dashboard and has not dismissed the intro
- **THEN** the dashboard SHALL render the assistant introduction with a link to agent settings
- **AND** dismissing it SHALL persist (client-side) and prevent re-showing
