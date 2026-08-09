## ADDED Requirements

### Requirement: Tickets are created from WhatsApp conversations or manually

The system SHALL support creating a ticket from an existing WhatsApp conversation and creating tickets manually from the CRM contact view. Each ticket SHALL reference the organization and optionally a contact and a conversation. Ticket creation SHALL be gated by the `tickets` module feature (`tickets_module`).

#### Scenario: Ticket created from a WhatsApp conversation

- **WHEN** an organization with the `tickets` module enabled creates a ticket from an open WhatsApp conversation with a contact
- **THEN** a ticket SHALL be persisted with status `open`, linked to the contact and conversation
- **AND** the ticket SHALL reference the creating member via `stytch_member_id`

#### Scenario: Ticket creation blocked without module

- **WHEN** an organization without the `tickets` module attempts to create a ticket
- **THEN** the system SHALL return HTTP 403 with a JSON body `{"error": "module_disabled", "module": "tickets"}`
- **AND** no ticket SHALL be created

### Requirement: Ticket lifecycle follows a defined state machine

The system SHALL enforce the ticket state machine: `open` → `in_progress` → `resolved`, with `waiting_customer` and `cancelled` as valid branches. `resolved` SHALL be terminal. Transitions SHALL be recorded as ticket events.

#### Scenario: Full lifecycle transition

- **WHEN** an assigned member moves a ticket from `open` to `in_progress`, then to `resolved`
- **THEN** the ticket status SHALL be `resolved`
- **AND** the ticket SHALL have transition events recorded for both transitions

#### Scenario: Invalid transition rejected

- **WHEN** a member attempts to move a ticket from `open` directly to `cancelled` without a valid branch (per policy) or from `resolved` to any other state
- **THEN** the system SHALL return HTTP 400 with a JSON error body
- **AND** the ticket status SHALL remain unchanged

### Requirement: Tickets support assignment to members

The system SHALL allow assigning a ticket to a member of the organization, referenced by `stytch_member_id`. Assignment SHALL require `ticket:manage` permission; viewing SHALL require `ticket:view`.

#### Scenario: Assign and unassign

- **WHEN** a member with `ticket:manage` assigns the ticket to another member
- **THEN** the ticket SHALL reflect the assignee `stytch_member_id`
- **AND** an assignment event SHALL be recorded

- **WHEN** the same member unassigns the ticket
- **THEN** the assignee SHALL be null
- **AND** an unassignment event SHALL be recorded

#### Scenario: Permission denied on assign

- **WHEN** a member with `ticket:view` but not `ticket:manage` attempts to assign a ticket
- **THEN** the system SHALL return HTTP 403

### Requirement: Ticket priorities and tags follow module configuration

The system SHALL restrict ticket priorities and tags to the sets configured in the organization's `tickets` module config (defaults: priorities `low`, `normal`, `high`; tags free-form). SLA due-at SHALL be computed on priority change as `created_at`-independent re-armed SLA based on the configured `sla_hours` per priority.

#### Scenario: Priority change re-arms SLA

- **WHEN** an organization with `sla_hours` 24 for `high` changes a ticket priority to `high`
- **THEN** the ticket's `sla_due_at` SHALL be set to now plus 24 hours
- **AND** a priority-change event SHALL be recorded

#### Scenario: Unknown priority rejected

- **WHEN** a member sets a ticket priority outside the configured priority set
- **THEN** the system SHALL return HTTP 400
- **AND** the priority SHALL remain unchanged

### Requirement: Internal notes are team-only

The system SHALL support internal (team-only) notes on tickets. Internal notes SHALL be visible only to organization members with `ticket:view` and SHALL never be transmitted to the customer-facing WhatsApp channel.

#### Scenario: Internal note visible to team

- **WHEN** a member with `ticket:manage` adds an internal note to a ticket
- **THEN** the note SHALL be persisted and visible to organization members with `ticket:view`

#### Scenario: Internal note never sent to customer

- **WHEN** an internal note is added and the ticket has an associated WhatsApp conversation
- **THEN** no message SHALL be sent to the customer via the WhatsApp bridge as a result of the note creation

### Requirement: Ticket events are append-only

The system SHALL record ticket lifecycle events (creation, status transitions, assignment changes, priority changes, note additions) as an append-only event history per ticket, including actor `stytch_member_id` and timestamp.

#### Scenario: Event history is complete and immutable

- **WHEN** a ticket has been created, assigned, re-prioritized, and resolved
- **THEN** the event history SHALL contain at least one event for each action in chronological order
- **AND** existing events SHALL NOT be modified or deleted by subsequent actions
