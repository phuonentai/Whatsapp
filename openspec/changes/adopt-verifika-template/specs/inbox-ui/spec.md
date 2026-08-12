# inbox-ui Delta Spec

## ADDED Requirements

### Requirement: Inbox renders Verifika messages-view composition

The inbox page (`app/dashboard/inbox`) SHALL render the Verifika `messages-view` composition: page header ("Mensajes" + subtitle) with a "Conectado" status badge and a "+ Nueva campaña" action linking to the existing campaigns flow; a stats row with icon-chip cards (conversaciones hoy, por responder, tasa de respuesta, tiempo promedio); a toolbar with search and channel/status filters; and the conversation list with avatar, label, snippet, time and unread badge. All composition SHALL render over the existing data logic (inbox store/queries, channel/status filters, unread indicators) — none of the behavior requirements in this spec SHALL change.

#### Scenario: Composition renders with real data

- **WHEN** an entitled user opens the inbox with existing conversations
- **THEN** the header, stats row and conversation list SHALL render in the Verifika composition with real inbox data

#### Scenario: Behavior requirements preserved

- **WHEN** the inbox renders in the new composition
- **THEN** replies, status/channel filters, quick replies, sequence mode, unread indicators and empty/failure states SHALL behave exactly as specified in this capability
