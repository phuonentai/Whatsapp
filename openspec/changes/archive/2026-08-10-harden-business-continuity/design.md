## Context

Business Continuity audit (`docs/business-continuity-audit.md`) flagged P0 risks. Current state:

- Postgres: single Docker volume, zero backups, zero restore procedure. RPO = ∞.
- WhatsApp webhooks: `ProcessWebhook` (handler) verifies signature, inserts `whatsapp.webhook_logs`, then synchronously publishes `whatsapp.message.received` / `whatsapp.message.echo` to `InMemoryEventBus` (`internal/platform/eventbus/bus.go:23`). Crash between commit and publish loses messages. `Publish` errors are logged and dropped (`webhook_service.go:87-113`). No delivery-level dedup (CRM row dedup exists via `ON CONFLICT`).
- Outbound Meta Graph API client (`internal/modules/whatsapp/infra/graphapi/client.go:94`): 15s timeout only — no retry, no breaker.
- Prod compose: no `migrate` job, no backend/Redis healthchecks; `/health` returns static OK in prod (`health.go:11-15`).
- `go-b2b-starter/app.env.bak` tracked in git; `.gitignore` misses `*.env.bak`.

Constraints: Clean Architecture (domain → app → infra), SQLC-generated persistence, Stytch B2B as runtime SSOT (no local credentials), OpenSpec governance.

## Goals / Non-Goals

**Goals:**
- Durable, at-least-once delivery of WhatsApp message events (no loss on crash; no duplicate downstream effects).
- Automated Postgres backups off-host with restore procedure and RPO ≤ 24h / RTO ≤ 4h.
- Meta Graph API resilience: breaker + backoff retry + durable retry queue.
- Production compose migration job + real health checks.
- Secrets hygiene: env backup files untracked, CI guard.

**Non-Goals:**
- Postgres replication / managed-DB HA (P1).
- Multi-instance horizontal scale-out of the backend (P1/P2) — though the outbox design makes it possible later.
- Alerting/monitoring stack (P1).
- Billing/invoicing pipeline changes.
- Local credential storage (SSOT invariant).

## Decisions

### D1. Postgres-backed outbox instead of Redis Streams or RabbitMQ

**Decision:** New `outbox_events` table in Postgres; dispatcher polls with `FOR UPDATE SKIP LOCKED`.

**Why:** Redis is currently a cache-only dependency and is fail-fast at startup (`redis/init.go` `log.Fatalf`); making message delivery depend on it inverts reliability. RabbitMQ is a new infrastructure dependency — no operational capacity to run it. Postgres is already the transactional store; atomic write of `webhook_logs` + outbox in one transaction is the core guarantee.

**Alternatives considered:** Redis Streams (rejected: Redis not durable-configured, fail-fast boot risk), RabbitMQ/Kafka (rejected: new infra, no ops), transactional outbox via Debezium (overkill at this scale).

### D2. Single outbox table shared by pipeline + send retry queue

**Decision:** One `outbox_events` table with `event_type`, `payload`, `status` (`pending|dispatched|dead_letter`), `attempts`, `next_attempt_at`, `last_error`. WhatsApp message events (received/echo) and Graph API send jobs both use it.

**Why:** One mechanism, one dispatcher, one retry/backoff policy, one dead-letter path. Send queue becomes a message-send event consumed by a sender handler — no separate queue infra.

**Alternative:** Separate send_queue table (rejected: duplicate machinery, no benefit at this volume).

### D3. Webhook delivery dedup keyed on provider webhook/message ID

**Decision:** Dedup on `(phone_number_id, webhook_id)` derived from the payload entry ID (Meta provides `entry[].id`) plus the set of `message[].id` values. Persisted in `whatsapp.webhook_logs` with a unique index; dedup check inside the same transaction as log + outbox insert.

**Why:** Meta retries after timeout/5xx must be acked without re-dispatch. `entry[].id` is stable per delivery batch. Message-level dedup complements existing CRM `ON CONFLICT` idempotency (already spec'd) — delivery dedup is earlier in the path, cheaper.

**Alternative:** Hash of full raw body (rejected: Meta retries can resend with re-ordered fields; message IDs are the stable business key).

### D4. Dispatcher: in-process goroutine with backoff, not a separate worker binary

**Decision:** Dispatcher runs in the API process (like the existing invoicing poller `invoicing/cmd/init.go:53`), polls every 1s, claims rows with `FOR UPDATE SKIP LOCKED`, retries with exponential backoff (base 1s, factor 2, cap 60s, jitter) up to 10 attempts, then `dead_letter`. Startup resumes pending events.

**Why:** No new deployment unit. Concurrency-safe for future multi-instance via row locking. Matches existing poller pattern in codebase.

**Risk accepted:** Dispatcher dies with process — events survive in DB, resume on restart (spec requirement).

### D5. Graph API circuit breaker + retry reuses existing breaker pattern

**Decision:** Implement breaker mirroring `internal/platform/stytch/circuit_breaker.go` (threshold/failure, timeout, half-open probe) wrapped around Graph API calls, plus per-send retry via outbox `attempts`/`next_attempt_at`. Fail-fast on open breaker → send job stays queued until breaker closes.

**Why:** Consistent with established codebase pattern; breaker (fast-fail, protects Meta) + durable retry (eventual delivery) are complementary. Permanent errors (401 invalid token) detected and dead-lettered without retry.

### D6. Backups: cron `pg_dump` script + R2 upload; no WAL archiving in this change

**Decision:** `scripts/backup_db.sh` — `pg_dump -Fc` → gzip → upload to R2 (existing `r2` module pattern) → prune by retention (default 14 daily). Local cron (or host cron) runs it. `scripts/restore_db.sh` restores to a fresh DB, verifies, then operator swaps. CI runs a restore drill against scratch DB.

**Why:** Logical dump is simple, version-agnostic, restorable cross-instance. WAL archiving (PITR, sub-24h RPO) requires postgres config + archive command wiring — defer to P1 alongside replication. RPO 24h matches audit target for this phase.

**Alternative:** WAL-E/pgBackRest (rejected: PITR complexity, defer to P1), managed RDS/Neon (rejected: vendor change, P1).

**Risk accepted:** Up to 24h data loss window; documented as contract limit.

### D7. Health: split liveness `/healthz` from readiness `/readyz`; prod compose wiring

**Decision:** `/healthz` = process liveness (always 200). `/readyz` = probes DB `Ping` + Redis `Ping`, returns 503 with dependency detail on failure. Caddy routes both to backend. Prod compose: `migrate` job (golang-migrate, same as dev), backend `depends_on` migrate + postgres/redis `service_healthy`, Redis gains healthcheck.

**Why:** Readiness truth for orchestration; liveness keeps process alive during recovery (spec: liveness stays 200). CI health-wait switches to `/readyz`.

### D8. Secrets hygiene

**Decision:** `git rm --cached go-b2b-starter/app.env.bak`, add `*.env.bak` to `.gitignore`, extend CI guard (existing mock-auth leak check at `ci.yml:26-31`) to scan tracked files for secret-shaped env patterns. Full history purge deferred: file contains test-env placeholders (`*-test-*`), rotation is defensive; a `git filter-repo` history rewrite is noted as follow-up if repo is publicized.

## Risks / Trade-offs

- [Outbox rows grow unbounded] → Retention: dispatched rows pruned after N days (configurable); dead-letter kept until operator replay/resolve.
- [Dedup index on large webhook_logs] → Index on `(phone_number_id, delivery_key)`; prune handled above; table is append-only audit anyway.
- [Dispatcher adds DB polling load] → 1s poll with `SKIP LOCKED` on small pending set; indexed on `status, next_attempt_at`.
- [Backup window vs live writes] → `pg_dump -Fc` is consistent; no locking; RPO gap is acknowledged contract.
- [Breaker + retry interplay: breaker open while sends queued] → Dead-letter only after both max attempts AND breaker closed (temporary errors don't burn attempts while breaker open — check breaker state before counting attempt).
- [R2 backup region availability] → Documented assumption; restore script supports alternate endpoint.
- [Behavior change in webhook path] → Existing spec scenarios kept intact; 200-after-commit changes timing only; Meta retry behavior unchanged (dupes now deduped).
- [gitignore guard false positives] → Guard scans for `secret-*`/`AKIA`-style patterns only, mirroring existing CI check style.

## Migration Plan

1. Add migrations: `outbox_events` table, `webhook_logs` dedup column + unique index.
2. Deploy backend with outbox code behind no feature flag (behavior change is the point); dispatcher starts on boot.
3. Existing in-memory bus remains for non-WhatsApp events; WhatsApp message events migrate to outbox (dual-publish during one release for observability is optional, not required).
4. Add backup scripts + cron, run one restore drill in staging before production cron enable.
5. Prod compose: migrate job, healthchecks, `/readyz`; CI health-wait updated.
6. `git rm --cached app.env.bak` + `.gitignore` + CI guard.

**Rollback (Git state):** Revert commit of archived change; outbox table + dedup index are additive (down migrations provided); rollback returns to in-memory bus behavior. **Rollback (Stytch tenant policy):** no Stytch policy changes in this change; outbox dispatcher reuses existing session verification — no tenant rollback required. **Data rollback:** if dispatcher misbehaves, stop it via env flag `OUTBOX_DISPATCHER_ENABLED=false` and keep webhook logs intact for replay (no loss, only delayed dispatch).

## Open Questions

- Dispatcher interval and max-attempt values: defaults (1s / 10 attempts / 60s cap) reasonable at launch volumes? Operator-configurable via env.
- Backup cron host: run inside prod compose as a sidecar container vs host cron — host cron chosen (no long-running container), confirm ops preference.
- Should outbound send retry reuse the same outbox table or get its own `message_send_queue` for clearer status semantics? (D2 favors shared; confirm at apply time.)
