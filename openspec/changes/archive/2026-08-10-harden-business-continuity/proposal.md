## Why

Business Continuity audit (`docs/business-continuity-audit.md`) found the platform is not continuity-ready: RPO is infinite (no Postgres backups), the core business asset — inbound WhatsApp messages — flows through an in-memory event bus and can be silently lost on crash, Meta webhook retries can duplicate events, and Meta Graph API outbound sends have no retry or circuit breaker. Single-host production compose has no DB migration job and liveness-only health checks that hide dependency outages.

## What Changes

- **BREAKING** Replace synchronous in-memory event publication for WhatsApp message events with a durable outbox: webhook handler persists webhook log + outbox entries atomically, returns 200, and a dispatcher delivers events asynchronously with retries and dead-letter recording.
- Add webhook idempotency: deduplicate processed webhook deliveries by webhook ID / message ID before event dispatch.
- Add outbound Meta Graph API resilience: circuit breaker, retry with exponential backoff, and durable retry queue for failed sends.
- Add Postgres backup automation: scheduled `pg_dump` to off-host storage plus documented restore procedure (RPO/RTO targets in spec).
- Add production readiness in compose: `migrate` job in production compose, backend + Redis healthchecks, and `/health` probes DB + Redis connectivity.
- Remove `app.env.bak` from git history; rotate any credentials it contained; ignore `*.bak` env files.

## Capabilities

### New Capabilities

- `data-backup-recovery`: Scheduled Postgres backups (dump + WAL archiving where feasible), off-host storage, restore procedure, and RPO/RTO contract.
- `durable-message-pipeline`: Outbox-based durable delivery of WhatsApp message events, async dispatch, retry with backoff, dead-letter recording, and replay.
- `whatsapp-provider-resilience`: Circuit breaker, retry policy, and durable queue for outbound Meta Graph API calls.
- `production-health-and-ops`: Health endpoints probing DB/Redis, production compose migration job and healthchecks, and secrets-hygiene enforcement for tracked env files.

### Modified Capabilities

- `whatsapp-webhook-ingress`: Add idempotent webhook processing requirement (dedup by delivery/message ID) and atomic persistence of webhook log + outbox entries before HTTP 200.

## Impact

- **Backend (Go)**: `internal/modules/whatsapp` (webhook service, Graph API client), `internal/platform/eventbus` (outbox dispatcher), new DB migrations (outbox table, dedup keys), server health handler, Redis role (optional, for dedup counters — primary dedup in Postgres).
- **DB**: New tables/migrations; existing `whatsapp.webhook_logs` gains dedup index and status transitions.
- **Ops**: `docker-compose.production.yml` (migrate job, healthchecks), `.github/workflows/ci.yml` (restore-drill test), `scripts/` (backup script, restore script), `.gitignore`, git history rewrite for `app.env.bak`.
- **Docs**: `docs/` runbook (backup, restore, RPO/RTO), `docs/business-continuity-audit.md` links to remediation.
- **Stytch B2B**: No API contract changes. Outbox/dispatch path reuses existing Stytch session verification for API requests. Identity data (Stytch) is NOT backed up by the local Postgres backup; fallback to cached-JWKS read-only mode remains unchanged per `stytch-authorization` spec. No Stytch tenant policy changes; Git-state rollback is via standard revert of the archived change.

## Non-Goals

- No local storage of credentials, passwords, MFA tokens, or session tokens (SSOT invariant unchanged — identity stays solely in Stytch B2B).
- No multi-instance/HA deployment, no Postgres replication or managed-DB migration in this change (tracked separately, P1).
- No replacement of the platform-wide in-memory event bus outside WhatsApp message events.
- No billing (Polar/MercadoPago) or invoicing (Siigo) pipeline changes.
- No alerting/monitoring stack (P1).

## Assumptions

- Meta delivers WhatsApp webhooks at-least-once; retries after non-2xx or timeout can duplicate deliveries (not verified against Meta docs in code; standard Cloud API behavior).
- Backup storage target is Cloudflare R2 (existing file storage) — bucket-level replication/durability of R2 itself is assumed, not verified.
- Rotating credentials found in `app.env.bak`: it contains Stytch test-env placeholders (pattern `*-test-*`), not production secrets; rotation is defensive.
