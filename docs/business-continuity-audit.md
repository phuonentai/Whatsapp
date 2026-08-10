# Business Continuity Audit Report

**Subject:** WhatsApp B2B platform (Go backend + Next.js frontend)
**Date:** 2026-08-10
**Auditors:** Business Continuity team
**Verdict:** **NOT CONTINUITY-READY.** Unit-level engineering is solid. System-level resilience is absent. The core business asset — inbound WhatsApp messages — flows through a volatile path with no durable queue. Data has no backup. Every component runs as a single instance.

---

## 1. Executive Summary

| Metric | Current State |
|--------|---------------|
| RPO | **∞** — no backups exist; any data loss is total and permanent |
| RTO | **Hours–days** — manual rebuild, manual migrations, no automation, no runbook |
| DR | **None** — no replica, no secondary region, no restore test ever run |
| HA | **None** — one instance of every component (Caddy, frontend, backend, Postgres, Redis) |
| Message durability | **None** — in-memory event bus, events lost on crash |

```
                    CURRENT POSTURE
┌────────────────────────────────────────────────────────┐
│  RPO:   ∞        RTO: hours–days      HA: none         │
│  DR:    none     Monitoring: liveness only             │
│  Message path:   HTTP webhook → sync → in-memory bus   │
└────────────────────────────────────────────────────────┘
```

A single lost container volume destroys all tenant data. A single failed deploy is a manual ops incident. A crash between DB write and event publish silently drops customer WhatsApp messages.

---

## 2. Scope and Method

- Read production/deployment configuration: `docker-compose.production.yml`, `docker-compose.yml`, `Caddyfile`, `setup.sh`
- Read backend internals: auth, whatsapp, billing, invoicing, files, event bus, server, redis, stytch platforms
- Read CI: `.github/workflows/ci.yml`
- Read governance state: `openspec list` (20 changes; 6 in-progress)
- Checked secret hygiene: git-tracked files, `.gitignore`

---

## 3. Findings by Domain

### A. Data Durability & DR — CRITICAL

| # | Finding | Evidence |
|---|---------|----------|
| A1 | **No backups anywhere.** Postgres is a named Docker volume. No `pg_dump` job, no WAL archiving, no PITR, no snapshot. Volume loss = total data loss. | `docker-compose.production.yml:95-96` |
| A2 | **No Postgres replication.** Single node, no standby, no failover, no streaming replica. | `docker-compose.production.yml:89-102` |
| A3 | **Restore never tested.** No restore procedure documented, no DR drill, no runbook in `docs/`. | `docs/` contains only `crm-design.md` + `dashboard.png` |
| A4 | **Redis persistence unconfigured.** `redis:alpine` default config, no AOF/RDB tuning. Redis holds JWKS cache + RBAC policy cache. Loss = cache miss + refetch (recoverable, but fail-fast risk below). | `docker-compose.production.yml:107-111` |
| A5 | **In-memory event bus.** `InMemoryEventBus` blocks on handlers, no outbox, no durable store, no replay. Events published after DB commit are lost on crash. | `go-b2b-starter/internal/platform/eventbus/bus.go:22-36` |
| A6 | **In-memory rate limiter.** Per-process token bucket; resets on restart; useless across instances; also fails open for distributed abuse. | `go-b2b-starter/internal/platform/server/middleware/ratelimit.go:9-14` |

### B. Availability & HA — CRITICAL

| # | Finding | Evidence |
|---|---------|----------|
| B1 | Single instance of everything: 1 Caddy, 1 Next.js, 1 Go API, 1 Postgres, 1 Redis — all on one host. Any host loss = full outage. | `docker-compose.production.yml` |
| B2 | **In-memory event bus forbids horizontal scaling.** Running 2+ backend instances breaks event delivery (handlers run in the instance that received the webhook only). Scale-out impossible without replacing the bus. | `bus.go:23` |
| B3 | **Health check is liveness-only.** `/health` returns static `OK` in prod — does not probe DB, Redis, or provider connectivity. Orchestrators and CI can see "healthy" while the app is unusable. | `go-b2b-starter/internal/platform/server/domain/health.go:11-15` |
| B4 | **Redis is fail-fast at startup.** `log.Fatalf` on Redis connect failure kills the process. A Redis blip during deploy = crash-loop. No degraded mode at boot. | `go-b2b-starter/internal/platform/redis/init.go:5-18` |
| B5 | Prod compose has no `restart` policy mismatch risk: `migrate` service absent from production compose. Migrations are manual, undocumented, and untested in prod. | `docker-compose.production.yml` (no migrate service) vs `docker-compose.yml:41-53` |
| B6 | Backend has no healthcheck in prod compose; `depends_on` for Redis is `service_started` (not healthy). | `docker-compose.production.yml:79-83` |

### C. Dependency Resilience — PARTIAL

| # | Finding | Evidence |
|---|---------|----------|
| C1 | **Stytch: good.** Two-tier circuit breaker (5 failures / 10s timeout / 2 half-open probes), Redis-backed JWKS cache with TTL, fallback to cached JWKS. | `go-b2b-starter/internal/platform/stytch/circuit_breaker.go`, `jwks_cache.go:71-110` |
| C2 | **OpenAI/LLM: good.** Circuit breaker + exponential backoff with jitter, permanent-error exclusion. | `go-b2b-starter/internal/platform/llm/infra/openai_client.go:343-410` |
| C3 | **Meta Graph API: NO retry, NO circuit breaker.** Message send path has a 15s timeout and nothing else. Graph API outage = failed sends with no backoff, no breaker, no queue. | `go-b2b-starter/internal/modules/whatsapp/infra/graphapi/client.go:94` |
| C4 | **Polar/Stripe, MercadoPago, Siigo:** webhook signature verification present; invoice poller acts as webhook fallback (good pattern). But no circuit breakers observed on these outbound call paths. | `invoicing/cmd/init.go:53-68`, `polar/webhook.go:26` |
| C5 | **Vendor concentration.** Auth (Stytch), billing (Polar + MercadoPago), invoicing (Siigo), messaging (Meta), AI (OpenAI). Each is a single point of failure with no abstraction-level fallback except the invoice poller. | — |

### D. Messaging Reliability (CORE BUSINESS) — CRITICAL

Inbound WhatsApp messages are the product. Current pipeline:

```
Meta webhook ──▶ HTTP handler ──▶ verify sig ──▶ DB insert (log)
     ──▶ parse ──▶ eventBus.Publish ──▶ in-memory handlers (sync, blocking)
```

| # | Finding | Evidence |
|---|---------|----------|
| D1 | **Synchronous processing in request path.** `ProcessWebhook` inserts log + publishes events before returning 200. Meta expects fast acks; handler block = Meta retries = duplicate risk. | `go-b2b-starter/internal/modules/whatsapp/app/services/webhook_service.go:43-117` |
| D2 | **Publish errors only logged, not retried.** If an event handler fails (DB down, downstream error), the message event is dropped. `Publish` returns error; service logs and continues. | `webhook_service.go:87-92, 108-113` |
| D3 | **No idempotency on webhook processing.** No dedup by `MessageSID`/webhook ID at processing layer. Meta at-least-once delivery + concurrent handlers = duplicate message events. Idempotency tests exist only for DB-level constructs, not the webhook path. | `webhook_service.go` (no dedup check); `internal/db/postgres/sqlc/integration/idempotency_test.go` |
| D4 | **No outbox / durable queue.** Business-critical events (message received) have zero durability. The `webhook_logs` table stores raw payloads ("for audit and replay") but no replay mechanism exists. | `models.go:634-637` |
| D5 | **Webhook log insert failure is swallowed** (logged, not fatal) — message still processed, but audit trail breaks. | `webhook_service.go:69-71` |
| D6 | **WhatsApp access tokens stored in DB** (`whatsapp_configs.access_token`). DB compromise = Meta API token compromise. No encryption at rest, no secret manager. | `handler.go:156`, `config.go:12` |

### E. Secrets & Security Hygiene

| # | Finding | Evidence |
|---|---------|----------|
| E1 | **`app.env.bak` is git-tracked.** `.gitignore` covers `app.env` but misses the `.bak`. Contains secret-shaped values (2 matches on `secret-test`/`project-test` patterns). | `git ls-files` shows `go-b2b-starter/app.env.bak`; `.gitignore` |
| E2 | Plaintext secrets in compose env + `.env` files (Stytch secret, Polar token, Postgres password). No secrets manager, no rotation policy. | `docker-compose.production.yml:4-9` |
| E3 | `NEXT_PUBLIC_POLAR_*` access tokens passed as build args → baked into client bundle. | `docker-compose.production.yml:44-45` |
| E4 | Postgres password shared verbatim across dev/prod templates. | `docker-compose.yml:11-12` |

### F. Observability — WEAK

| # | Finding | Evidence |
|---|---------|----------|
| F1 | Custom logger only; no metrics, no tracing, no structured telemetry export, no centralized log collection. | `go-b2b-starter/internal/platform/logger` |
| F2 | No alerting. No SLO/SLI definitions. No on-call. No runbook. Outage detection = customer complaint. | — |
| F3 | Health endpoint hides dependency state in prod. | `health.go:12-14` |

### G. Deployment & Release — WEAK

| # | Finding | Evidence |
|---|---------|----------|
| G1 | **CI only, no CD.** `.github/workflows/ci.yml` tests but nothing deploys. Deploy = manual `docker compose` on a host. No staging environment. | `ci.yml` |
| G2 | **No rollback automation.** No image versioning strategy, no blue/green, no DB migration rollback drill. `make migratedown` exists but prod procedure undocumented. | — |
| G3 | Migration versioning churn: `fix-migration-renumber` change in-progress (16/18) — risk of prod migration drift while active. | `openspec list` |
| G4 | CI e2e runs on `pg16` while prod compose uses `pg17` — version skew between tested and deployed DB. | `ci.yml:83` vs `docker-compose.production.yml:90` |

### H. Governance / Change Risk

| # | Finding | Evidence |
|---|---------|----------|
| H1 | 6 active changes in progress, several large: `add-mercadopago-billing` (34/48), `add-ci-pipeline` (16/21), `add-siigo-invoicing` (20/24), `add-whatsapp-embedded-signup` (18/20). High change velocity on billing + messaging = elevated regression risk. | `openspec list` |
| H2 | Git history shows day-scale mega-commits ("day-long expansion") — change size obscures auditability, complicates bisect during incidents. | `git log` |

---

## 4. What Is Done Well (acknowledged)

- Webhook signature verification on all providers (Polar, Siigo, WhatsApp X-Hub-Signature-256)
- Stytch circuit breaker + JWKS cache fallback, matching governance spec
- OpenAI client: breaker + backoff + jitter
- Invoice poller as webhook fallback (polling safety net pattern)
- Graceful shutdown on SIGINT/SIGTERM with 5s timeout
- CI with backend tests, frontend lint/build, offline e2e, migration-uniqueness check, mock-auth leak guard
- Idempotency-tested DB constraints; tenant isolation migrations
- File assets on Cloudflare R2 (off-host object storage)
- Webhook raw payload logging (replay capability exists, unused)

---

## 5. Risk Matrix

| # | Risk | Likelihood | Impact | Level |
|---|------|-----------|--------|-------|
| 1 | Host/volume loss → total data loss | Medium | Catastrophic | **CRITICAL** |
| 2 | Crash between DB write & event publish → lost WhatsApp messages | Medium | High | **CRITICAL** |
| 3 | Duplicate webhook processing (Meta retries) | High | Medium | HIGH |
| 4 | Graph API outage → failed sends, no backoff | Medium | High | HIGH |
| 5 | Redis blip at boot → crash-loop | Medium | Medium | MEDIUM |
| 6 | `app.env.bak` leak in repo | Low | High | MEDIUM |
| 7 | DB compromise → WhatsApp tokens + secrets | Low | Catastrophic | MEDIUM |
| 8 | Deploy fails mid-migration → manual recovery | Medium | High | MEDIUM |
| 9 | LLM provider outage | Medium | Medium | MEDIUM |
| 10 | Billing webhook loss (Polar/MP) | Low | Medium | LOW (invoice poller mitigates) |

---

## 6. RPO / RTO Targets vs Current

| Component | Target | Current |
|-----------|--------|---------|
| Customer data (Postgres) | RPO ≤ 15 min, RTO ≤ 4 h | RPO ∞, RTO hours–days |
| Inbound messages | at-least-once, never dropped | may drop on crash |
| Full platform availability | 99.9% | single host, no HA |
| Restore drill | quarterly, tested | never run |

---

## 7. Recommendations

### P0 — Do this week (existential risk)

1. **Backup Postgres now.** Daily `pg_dump` + WAL archiving (or managed DB) to off-host storage (R2/S3). Test restore. Document RPO/RTO.
2. **Durable message pipeline.** Replace in-memory bus for message events with durable queue (Postgres outbox table + dispatcher, or Redis Streams/RabbitMQ). Webhook handler: persist event atomically with log row → ack 200 → async dispatch. Enables retry + replay + horizontal scale.
3. **Webhook idempotency.** Dedup by webhook/event ID (MessageSID) at processing layer, transaction-isolated.
4. **Graph API resilience.** Circuit breaker + retry with backoff on message send; queue failed sends for retry.
5. **Remove `app.env.bak` from git history.** Rotate any credentials it contains. Add `*.bak` to `.gitignore`.
6. **Production healthcheck.** Make `/health` probe DB + Redis; add healthchecks to backend/redis in prod compose; add `migrate` job to prod compose.

### P1 — This month

7. **Postgres replication** (streaming standby or managed HA) for failover.
8. **Redis persistence** (AOF) + make Redis degradation non-fatal (cache-off mode instead of `log.Fatalf`).
9. **CD + rollback.** Image tags, staging env, deployment automation, migration rollback drill.
10. **Alerting + metrics.** Export metrics (provider latency, breaker state, webhook backlog, dead-letter count); alert on circuit-open, webhook failure rate, backup failure.
11. **Encrypt WhatsApp tokens + secrets** at rest; move to secrets manager; rotate quarterly.
12. **Rate limiter → shared store** (Redis) or per-tenant buckets.

### P2 — Quarter

13. Multi-instance deployment enabled by durable bus; region redundancy for DR.
14. Message replay UI/tooling on `webhook_logs` raw payloads.
15. Restore drills scheduled; DR runbook written; chaos tests (kill Redis, kill DB volume, stop Graph API).
16. Align CI DB version with prod (pg17); add migration-foward-only guard in CD.

---

## 8. Conclusion

The platform is engineered at function level, not system level. Strengths (circuit breakers, signature verification, poller fallbacks, CI) show the authors know resilience patterns — but none of them cover the two things that actually kill the business: **data loss** and **message loss**. Until backups exist and the message pipeline is durable, the business is one host crash away from losing its customers' data and one process crash away from silently losing conversations.

**Business continuity team recommends: proceed with P0 items as blocking; do not scale customer acquisition on the current single-node, no-backup posture.**
