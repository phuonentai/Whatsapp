# ISO/IEC 27001:2022 — Security Officer Report

**Subject:** Business-continuity remediation change `harden-business-continuity`
**Author:** Information Security Officer (role: security officer)
**Date:** 2026-08-10
**Companion:** `docs/business-continuity-audit.md` (pre-change audit), `docs/runbook-backup-restore.md` (operational procedure)

---

## 1. Executive Summary

The change implements the P0 findings of the Business Continuity audit. From an ISO 27001:2022 standpoint, the organization materially reduced exposure in two of the highest-severity domains: **data integrity/availability** (Annex A 8.13, 8.14) and **information leakage / secure development** (A 8.12, 8.25, 8.31). The message-ingestion path — the core revenue asset — is now at-least-once durable instead of fire-and-forget.

**Verdict:** Acceptable residual risk for the current operating phase, with a documented, dated remediation backlog (Section 6). The platform is NOT yet at a target SoA state for production-critical services (no HA, no PITR, no monitoring) — see residual risks.

---

## 2. Scope of This Report

Assessment of the implemented change `harden-business-continuity` against ISO/IEC 27001:2022 Annex A controls. Not a full certification audit: no ISMS governance layer, policy set, or certification body review exists yet. This report treats the organization as implementing controls for a future SoA.

Implemented scope:
- Durable outbox for WhatsApp message events (at-least-once, retry/backoff, dead-letter, replay)
- Webhook delivery deduplication (idempotent ingest)
- Durable outbound send queue with circuit breaker + retry
- Postgres backup automation + restore procedure + CI restore drill (RPO ≤ 24h, RTO ≤ 4h)
- Production readiness: DB/Redis probing health endpoints, prod compose migration job + healthchecks
- Secret hygiene: tracked `.env.bak` removed, ignore rules, CI secret-scan guard

---

## 3. Risk Statement (Pre- vs Post-Change)

| # | Risk | Pre-change | Post-change |
|---|------|-----------|-------------|
| R1 | Total data loss (volume failure) | **Critical** — no backups, RPO ∞ | Medium — daily off-host backups, restore drills, RPO 24h |
| R2 | Silent loss of inbound WhatsApp messages (crash between commit and dispatch) | **Critical** — in-memory bus | Controlled — outbox, resume on restart, replay from `webhook_logs` |
| R3 | Duplicate processing on Meta retries | High | Low — delivery dedup + existing `ON CONFLICT` idempotency |
| R4 | Meta Graph API outage → failed sends | High | Low/Medium — breaker + durable retry queue |
| R5 | Secret leakage via tracked env backup files | Medium | Low — removed, ignored, CI-guarded |
| R6 | Deploy without migrations / blind health | High | Low — prod migrate job, `/readyz` probes |
| R7 | Token/credential compromise via DB breach | Medium (unmitigated) | Medium (unchanged — P1: encrypt at rest + secrets manager) |
| R8 | Undetected outage (no alerting) | High (unmitigated) | High (unchanged — P1: monitoring/alerting) |

---

## 4. Annex A Control Mapping (Implemented / Partially Implemented)

### A 8.13 — Information backup — **IMPLEMENTED**

- Daily logical backup (`scripts/backup_db.sh`) to Cloudflare R2, off-host, with SHA-256 checksums.
- Retention pruning (14 days), previous backup never deleted on failure.
- Restore procedure (`scripts/restore_db.sh`) targeting a fresh DB, never the live one; integrity verification (row counts).
- Automated restore drill in CI (`backup-drill` job) — evidence of restorability, not just backup existence.
- RPO ≤ 24h / RTO ≤ 4h contract defined in spec `data-backup-recovery` and runbook.

**Evidence:** `.github/workflows/ci.yml` (backup-drill job), `go-b2b-starter/scripts/backup_db.sh`, `restore_db.sh`, `docs/runbook-backup-restore.md`, `openspec/specs/data-backup-recovery/spec.md`.

**Residual:** no point-in-time recovery (PITR); up to 24h loss window. Accepted; P1 for WAL archiving.

### A 8.14 — Redundancy of information processing facilities — **PARTIALLY IMPLEMENTED**

- Readiness endpoint `/readyz` probes Postgres + Redis; liveness `/healthz` independent — correct signalling for orchestration (prevents killing a process mid-recovery; enables failover later).
- Redis switched to AOF persistence in prod compose (cache state survives restart).
- Backend now waits on `migrate` job + healthy dependencies in prod compose.

**Residual (blocking full control):** single host, single instance of every component, no DB replica, no multi-AZ. Rated P1. The outbox design (transaction-isolated `FOR UPDATE SKIP LOCKED` claims) makes multi-instance deployment possible without redesign — a prerequisite condition for redundancy now exists.

### A 8.12 — Data leakage prevention — **IMPLEMENTED**

- `go-b2b-starter/app.env.bak` removed from tracking (staged deletion; history purge deferred — file contained test-env placeholders, low exposure; full `git filter-repo` rewrite scheduled if repo becomes public).
- `.gitignore` now blocks `*.env.bak` / `*.env.bak.*`.
- CI guard fails the pipeline on: (a) secret-shaped values in tracked env files (with placeholder whitelist to avoid false positives on templates), (b) any tracked `.env.bak`.

**Evidence:** `.github/workflows/ci.yml` step "Verify no secret-bearing env files are tracked", `.gitignore` lines 11–19.

**Residual:** secrets remain plaintext in host env files/process env (P1: secrets manager + rotation). WhatsApp Graph API tokens stored plaintext in `whatsapp.whatsapp_configs` (P1: encryption at rest). History purge pending.

### A 8.25 — Secure development lifecycle — **IMPLEMENTED (extended)**

- Behavioural changes gated by OpenSpec: proposal → specs → design → tasks → verification gate → archive; deltas folded into living specs (`whatsapp-webhook-ingress` modified; four new capability specs).
- Verification gate executed: build, vet, 29 backend test packages, frontend lint/build, 103/103 e2e, migration apply on fresh DB, backup/restore drill.

### A 8.29 — Security testing in development & acceptance — **IMPLEMENTED (extended)**

- New negative-path tests: duplicate delivery ack without re-dispatch, atomic rollback on failure, breaker open/close/half-open, dead-letter exhaustion, restart resume, replay, send-state transitions.
- CI: backend unit, frontend lint+build, offline e2e, migration-uniqueness check, mock-auth confinement check, secret-scan guard, backup drill.

### A 8.31 — Separation of development, test and production environments — **PARTIALLY IMPLEMENTED**

- Prod compose (`docker-compose.production.yml`) now diverges correctly: migrate job, healthchecks, AOF Redis.
- CI guard rejects `AUTH_MOCK_ENABLED=true` in production configuration; e2e runs only with mock auth against `saas_db_test`.

**Residual:** no dedicated staging environment (P1). CI DB is pg16, prod compose pg17 — version skew tracked (P2).

### A 8.16 — Monitoring activities — **PARTIALLY IMPLEMENTED**

- Readiness/liveness endpoints give orchestrators and operators dependency truth.
- Outbox exposes dead-letter state; replay endpoint (`POST /api/v1/whatsapp/config/logs/:id/replay`) gives a recovery path for event loss; `webhook_logs` remains an audit trail.

**Residual (blocking full control):** no metrics export, no alerting, no SLOs, no centralized log collection — P1. Backup failure currently surfaces only in logs/cron exit codes (runbook documents the signal; alerting needed).

### A 6.8 — Information security event management — **PARTIALLY IMPLEMENTED**

- Dead-letter + replay + `webhook_logs` provide the audit trail and recovery mechanism for message-pipeline incidents.
- **Residual:** incident response runbook (severity levels, on-call, escalation) not yet documented — P1.

### A 5.2 / A 5.8 — Roles & responsibilities / ICT readiness — **PARTIALLY IMPLEMENTED**

- RPO/RTO contract defined (A 5.8-adjacent: information security aspects of business continuity now stated and spec'd).
- **Residual:** full BIA/BCP, tested DR drills on a schedule (not just CI), and continuity exercises — not established (P2).

---

## 5. Controls Explicitly NOT in Scope (Change Non-Goals)

| Control | Status | Rationale |
|---------|--------|-----------|
| A 8.5 Authentication (Stytch SSO/RBAC) | Unchanged | Runtime SSOT; existing `stytch-authorization` spec governs; no change needed for these risks |
| A 8.24 Cryptography at rest/transit | Unchanged | TLS via Caddy; at-rest encryption of WhatsApp tokens = P1 |
| A 8.28 Secure coding | Unchanged | Existing practice; covered by A 8.25/8.29 extensions |
| A 5.29 Security in supplier relationships | Unchanged | Vendor risk (Meta/Stytch/Polar/OpenAI) documented in BC audit; vendor resilience work is P1/P2 |

---

## 6. Residual Risks & Remediation Backlog (Dated)

| Priority | Item | Control impact | Owner |
|----------|------|----------------|-------|
| P1 | Postgres replication / managed DB HA; WAL archiving (PITR) | A 8.13, A 8.14 | Ops |
| P1 | Monitoring, metrics, alerting, SLOs; backup-failure alerting | A 8.16, A 6.8 | Ops |
| P1 | Secrets manager + credential rotation; encrypt WhatsApp tokens at rest | A 8.12, A 8.24 | Sec |
| P1 | Staging environment; CD with rollback; CI DB version = prod (pg17) | A 8.31 | Eng |
| P1 | Incident response runbook + on-call | A 6.8 | Sec/Ops |
| P2 | Restore drills quarterly + full DR exercise (BIA/BCP) | A 5.8 | Ops |
| P2 | Git history purge of `app.env.bak` (if repo publicized) | A 8.12 | Sec |

---

## 7. Conclusion & Recommendation

The change converts the two existential risks (data loss, message loss) into controlled, auditable, recoverable states. Security officer position:

1. **Accept** the current residual risk for the existing operating phase, **on condition** that the P1 backlog (HA/replication, alerting, secrets management) is executed before significant customer-base growth.
2. **Require** the backup runbook to be executed and a restore drill logged quarterly from this date.
3. **Require** monitoring/alerting (P1) before declaring the service production-ready per a future SoA.
4. Recommend starting a formal ISMS-scoping exercise (SoA, risk treatment plan, BIA) as the next governance step toward ISO 27001:2022 certification.

Signed — Security Officer, 2026-08-10.
