# ai-ux-affordances Specification

## Purpose
TBD - created by archiving change ai-ux-polish. Update Purpose after archive.
## Requirements
### Requirement: AI chat messages render markdown

Knowledge-chat assistant messages SHALL render GitHub-flavored markdown (headings, lists, bold, italic, code blocks, links) via a safe renderer, SHALL NOT render raw markdown syntax as text, and SHALL sanitize HTML to prevent injection. Each assistant message SHALL offer a copy action for its text.

#### Scenario: Assistant message renders markdown structure

- **WHEN** the assistant responds with a message containing a list and a bold phrase
- **THEN** the message SHALL render the list as list items and the phrase as bold, not raw syntax

#### Scenario: Code block renders monospaced and copyable

- **WHEN** the assistant response contains a fenced code block
- **THEN** it SHALL render monospaced with a copy button

#### Scenario: Raw HTML is not executed

- **WHEN** an assistant response contains an HTML tag
- **THEN** it SHALL render as escaped text and SHALL NOT execute

### Requirement: Citations show document titles

Knowledge-chat citation cards SHALL display the referenced document's title (joined from the documents query), an icon reflecting the document type, and the similarity context; the raw "Document #id" label SHALL NOT be the primary label. Citations for unknown/removed documents SHALL render a graceful fallback label.

#### Scenario: Citation card shows document title

- **WHEN** an assistant response references a known document
- **THEN** the citation card SHALL display the document's title and a document icon

#### Scenario: Unknown document shows fallback

- **WHEN** a cited document is no longer in the workspace
- **THEN** the citation SHALL render a fallback label (e.g., "Documento no disponible") instead of a raw internal id

### Requirement: AI output renders as structured UI

The CRM AI audience builder SHALL render its result as a structured card (segment criteria as labeled chips or lists, estimated audience size, consent-exclusion notice) with accept, edit, and regenerate actions. Raw JSON output SHALL NOT be the primary presentation.

#### Scenario: Audience result renders as structured card

- **WHEN** the AI audience builder returns a result
- **THEN** the result SHALL render criteria as labeled chips/lists, the audience size as a number, and the consent-exclusion notice as a banner
- **AND** actions SHALL offer accept, edit, and regenerate

#### Scenario: Raw JSON not shown

- **WHEN** the audience result renders
- **THEN** the raw JSON payload SHALL NOT be the primary display

### Requirement: Suggestion panel shows loading and per-item pending states

The inbox agent-suggestion panel SHALL render a skeleton while pending suggestions load (no full-panel `null` flash), and SHALL track pending state per suggestion so an in-flight approve/reject disables only that suggestion's actions.

#### Scenario: Panel loads with skeleton

- **WHEN** the pending-suggestions query is loading
- **THEN** the panel SHALL render a skeleton placeholder instead of disappearing

#### Scenario: Approving one suggestion disables only its actions

- **WHEN** a user approves one of several pending suggestions
- **THEN** only that suggestion's approve/reject actions SHALL show pending state; the others SHALL remain interactive

### Requirement: Streaming assistant text is announced

Knowledge-chat assistant streaming SHALL be contained in a live region (`aria-live="polite"`) so screen readers announce the response as it streams; the stream SHALL NOT spam announcements per token.

#### Scenario: Screen reader receives streamed response

- **WHEN** the assistant response streams tokens
- **THEN** the live region SHALL announce the accumulating text without per-token spam

### Requirement: Suggestion panel offers conversation context

Before approving, the user SHALL be able to expand the source conversation context behind a suggestion, rendered as a read-only thread excerpt.

#### Scenario: User expands context before approving

- **WHEN** a suggestion is pending and the user expands its context control
- **THEN** the panel SHALL show the read-only conversation excerpt the suggestion was based on

