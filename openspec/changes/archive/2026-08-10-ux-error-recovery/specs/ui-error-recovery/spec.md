## ADDED Requirements

### Requirement: Data-loading views render error and retry states

Every data-loading view (CRM lists, tickets, inbox conversation list, settings sections, knowledge document list) SHALL distinguish loading, error, and success states. When a data query fails, the view SHALL render an inline error state with a retry action instead of an indefinite loading indicator. The system SHALL provide a shared error-state component with a retry button reused across views.

#### Scenario: Failed contact query shows error with retry

- **WHEN** the contacts query fails (e.g. 5xx or network error)
- **THEN** the Contactos view SHALL render an error state with a "Reintentar" action instead of "Cargando contactos..."
- **AND** clicking retry SHALL re-run the query

#### Scenario: Failed ticket query shows error with retry

- **WHEN** the ticket list query fails
- **THEN** the tickets view SHALL render an error state with a retry action instead of an indefinite "Cargando..."

#### Scenario: Inline settings error keeps page usable

- **WHEN** a settings section query fails
- **THEN** the section SHALL render its error alert with retry while the rest of the page remains interactive

### Requirement: Mutations never fail silently

Every user-triggered mutation (send message, save, create, delete, toggle) SHALL surface a visible error on failure: a toast, an inline message, or both. A failed mutation SHALL NOT leave the UI in a state that falsely suggests success, SHALL NOT produce an unhandled promise rejection, and SHALL preserve any user-entered input that was not persisted.

#### Scenario: Failed send keeps draft and shows toast

- **WHEN** a user sends a reply and the send request fails
- **THEN** an error toast SHALL be shown, the draft SHALL remain in the input, and no message SHALL appear in the thread

#### Scenario: No unhandled rejections on failure

- **WHEN** any mutation rejects
- **THEN** the rejection SHALL be handled by the UI layer (toast or inline error) and SHALL NOT surface as an unhandled promise rejection

### Requirement: Product flows do not use native browser dialogs

Product flows SHALL NOT use `window.alert`, `window.confirm`, or `window.prompt`. Destructive or blocking confirmations SHALL use the existing custom `ConfirmDialog` component, and blocking notices SHALL use toasts or inline banners.

#### Scenario: Plan switch uses custom dialog

- **WHEN** a user attempts to switch plans while an active subscription exists
- **THEN** the notice SHALL render in the custom dialog component and SHALL NOT use `window.alert`

#### Scenario: Member role change uses custom dialog

- **WHEN** an admin changes a member's role
- **THEN** confirmation SHALL use the custom dialog and SHALL NOT use `window.confirm`

#### Scenario: Compliance forget uses custom dialog

- **WHEN** a user triggers the compliance forget flow
- **THEN** confirmation SHALL use the custom dialog and SHALL NOT use `window.confirm`

### Requirement: Unread inbound messages are indicated in the conversation list

The inbox conversation list SHALL show an unread indicator (count or dot) on conversations that have new inbound messages not yet seen by the current user, and SHALL clear the indicator when the conversation is opened or the user replies.

#### Scenario: New inbound message marks conversation unread

- **WHEN** a conversation receives a new inbound message and the user has not opened it
- **THEN** the conversation item SHALL display an unread indicator

#### Scenario: Opening the conversation clears the indicator

- **WHEN** the user opens a conversation with an unread indicator
- **THEN** the indicator SHALL be cleared for that conversation

### Requirement: Live updates are announced to assistive technology

Live regions SHALL announce message arrival in the open conversation thread, streaming assistant text in the knowledge chat, and incoming agent suggestions, using `aria-live`/`role="status"` where supported.

#### Scenario: Incoming message announced in thread

- **WHEN** a new message arrives in the open conversation
- **THEN** the thread container SHALL announce it via a live region (`role="log"` or `aria-live="polite"`)

#### Scenario: Streaming assistant text announced

- **WHEN** the knowledge chat streams an assistant response
- **THEN** the assistant message container SHALL be announced via a live region as tokens arrive
