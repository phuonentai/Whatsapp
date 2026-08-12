## Context

Audit of `go-b2b-starter/` (Go 1.25, Gin, Clean Architecture + dig DI) found:

- `jwks_cache.go:20` caches Stytch JWKS keys for 24h, violating the constitution's
  `TTL <= 300s` cryptographic-verification invariant (`openspec/config.yaml`).
- No backend response compression; Next.js `compress:true` only covers frontend.
- `SetupPrometheus` exists but is never called; no `/metrics` route, no request metrics.
- No pprof registration.
- `internal/modules/auth` is a god-dependency: 27 files / 142 call sites use
  `auth.GetRequestContext`/`GetIdentity`/`GetOrganizationID`/etc. from 9+ modules.
- `billing/handler.go` imports `infra/polar` + `infra/mercadopago` for webhook signature
  verification; `invoicing/handler.go` imports `infra/siigo` and its `Config`. HTTP layer
  reaches into infra, violating `handler -> app -> domain` direction.

## Goals / Non-Goals

**Goals:**
- Restore JWKS cache TTL to ≤300s per constitution.
- gzip compression for compressible backend responses; never buffer the SSE chat stream.
- Wire a real Prometheus `/metrics` endpoint with request counters and latency histograms.
- Register pprof behind an explicit enable flag (off in prod by default).
- Extract request-context accessors into `internal/platform/authcontext` so modules read
  identity/org context without importing the auth module.
- Move webhook signature verification behind domain interfaces in billing and invoicing.
- Preserve all behavior; this is structural + compliance work only.

**Non-Goals:**
- No change to Stytch RBAC policy resolution, session handling, or permission semantics.
- No local credential storage (constitution: PostgreSQL stores only stytch_member_id /
  stytch_organization_id FKs).
- No change to the RBAC policy Redis cache (already 5min, compliant).
- No OTel migration (ROADMAP Q4), no frontend polling changes, no new DB schema.

## Decisions

### 1. JWKS TTL: 24h → 5min (one-line const change)

`jwks_cache.go:20`: `jwksCacheTTL = 5 * time.Minute`. Matches the RBAC policy cache TTL
(`rbac_policy.go:23`) and `config.yaml` invariant. Alternative considered: make TTL
configurable — rejected, spec pins ≤300s; a const keeps it auditable.

### 2. gzip: custom stdlib middleware, SSE + metrics excluded

New `internal/platform/server/middleware/compression.go` using `compress/gzip`.
- Compress when: request `Accept-Encoding` includes gzip, response content-type is
  compressible (json/text/html), path not excluded, no existing `Content-Encoding`.
- Excluded: `/api/example_cognitive/chat` (SSE streaming), `/metrics`.
- Registered in `setupMiddleware()` (`middleware.go:16`) after RequestID, before logging.

Alternative considered: `gin-contrib/gzip` — rejected (v0.0.6 is old, adds dep, offline
risk). SSE + gzip buffering breaks token streaming — exclusion is mandatory.

### 3. Prometheus: wire existing setup + request metrics middleware

- Call `metrics.SetupPrometheus(engine)` at boot (in `api.Init` after route registration).
- New metrics middleware registering `http_requests_total{method,status,path}` counter and
  `http_request_duration_seconds{method,path}` histogram; skips `/metrics` and SSE path.
- Keep `/metrics` unauthenticated (standard for scraping); gated only by being unregistered
  when not wired — default wired. Alternative considered: full OTel — rejected (Q4 scope).

### 4. pprof: gated by `PPROF_ENABLED`

Register `net/http/pprof` under `/debug/pprof` on the Gin engine when
`PPROF_ENABLED=true`. Default false; forced off in prod unless explicitly set.

### 5. Auth context seam: `internal/platform/authcontext`

New package owns:
- Types: `Identity`, `RequestContext` (moved from `auth.go`).
- Context keys + accessors (moved from `context.go`): `Set/Get/MustGet Identity`,
  `Set/Get/MustGet RequestContext`, `GetOrganizationID`, `GetAccountID`,
  `WithIdentity`, `IdentityFromContext`, `WithRequestContext`, `RequestContextFromContext`.

`auth` middleware (`middleware.go`) writes via `authcontext.SetIdentity` /
`authcontext.SetRequestContext`. `auth` package re-exports moved symbols for its own
adapter code (`adapters/stytch`). `OrganizationEntity`/`AccountEntity` stay in `auth`
(used only by bootstrap lookup adapters — 2 sites).

Migration: 27 files switch `auth.GetRequestContext` → `authcontext.GetRequestContext`,
etc. Files using both context reads and `RequirePermissionFunc` keep both imports.

Alternative considered: keep accessors in `auth` and have modules depend on a slim
interface — rejected: Go has no partial-package imports; the only true decoupling is
moving the symbols to a platform package.

### 6. Webhook verifier seams

`billing/domain`:
```go
type WebhookVerifier interface {
    VerifyPolar(payload []byte, msgID, msgTimestamp, signature, secret string) error
    VerifyMercadoPago(payload []byte, signature, secret string) error
}
```
Header-name constants move to `billing/domain/webhook.go`. Infra adapters implement it;
DI provides a composite verifier; `NewHandler` depends on the interface.

`invoicing/domain`:
```go
type WebhookVerifier interface {
    Verify(payload []byte, signature, secret string) error
}
```
`invoicing/infra/siigo` implements it; handler takes verifier + `sandbox bool` +
`webhookSecret string` instead of `*siigo.Config`.

Rationale: mirrors the existing `ProviderRouter` pattern (Polar→MercadoPago) already in
the codebase; keeps signature algorithms (SVIX/HMAC) inside infra where the crypto libs live.

## Risks / Trade-offs

- [27-file import migration] → mechanical renames; build gate after each module group
  (CRM/WhatsApp/Instagram first, then the rest); final `grep` assertion that zero context
  reads remain outside auth.
- [gzip + SSE buffering] → explicit path exclusion for `/api/example_cognitive/chat`;
  tested with a streaming request.
- [gzip + metrics interplay] → metrics middleware runs before compression so histogram
  records unencoded size; both skip `/metrics` and the SSE path.
- [JWKS TTL 5min increases Stytch fetch rate] → bounded: one fetch per new kid per 5min
  window; stale-key exposure window shrinks from 24h to ≤5min (compliance wins).
- [Webhook verifier interface churn] → existing `polar.VerifyWebhookSignature` and
  `mercadopago.VerifyWebhookSignature` keep their tests; adapters wrap, not replace.
- [Prometheus dep already in go.mod (v1.20.5)] → no new dependency; wire the dead code.

## Migration Plan

1. Implement each task in order (see tasks.md). Each task is build/test-gated.
2. Deploy: single commit per task; backend redeploy applies gzip/metrics/pprof config.
3. Rollback: `git revert` per task commit. No DB migration. JWKS cache self-heals in ≤5min
   after revert. No Stytch tenant-policy change (config- and cache-only).

## Open Questions

- Should `/metrics` be exposed on a separate internal port to avoid exposing request
  counts to the public? Default: keep on main router (standard Prometheus scrape); revisit
  if a separate metrics port is required by infra.
