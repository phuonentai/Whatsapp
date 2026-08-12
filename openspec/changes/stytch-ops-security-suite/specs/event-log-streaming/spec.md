## ADDED Requirements

### Requirement: Auth event streaming to observability destinations

The system SHALL stream Stytch auth events to configured observability destinations via Stytch event-log streaming (free beta), configured in the Stytch dashboard: Datadog (site + API key) and/or Grafana Loki (public `/loki/api/v1/push`, gzipped JSON). The configuration SHALL be documented in `STYTCH_CONFIGURATION.md` including the event taxonomy, the destinations, and disable/rotate steps.

#### Scenario: Events stream to Datadog

- **WHEN** the Datadog destination is configured with a valid site and API key
- **THEN** Stytch auth events SHALL appear in the Datadog logs

#### Scenario: Events stream to Grafana Loki

- **WHEN** the Loki destination is configured with a reachable `/loki/api/v1/push` endpoint
- **THEN** Stytch auth events SHALL appear in Loki as gzipped JSON

#### Scenario: Destination disabled cleanly

- **WHEN** a streaming destination is disabled in the dashboard
- **THEN** no further events SHALL be streamed to it
- **AND** the application SHALL be unaffected

### Requirement: No application code for streaming

Event streaming SHALL be dashboard-configured only. No application code SHALL send events to the destinations, and no Stytch SDK streaming call SHALL be introduced.

#### Scenario: Streaming is configuration-only

- **WHEN** event streaming is enabled
- **THEN** no code path in the application SHALL push events to the destinations
