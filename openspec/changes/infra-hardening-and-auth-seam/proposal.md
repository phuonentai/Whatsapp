## Why

Two problem clusters surfaced in a codebase audit:

1. **Infrastructure gaps.** The JWKS public-key cache TTL is 24h (`internal/modules/auth/adapters/stytch/jwks_cache.go:20`) while the SSOT constitution mandates `TTL <= 300s` — a spec violation that can keep stale Stytch keys valid for a full day. The backend has no response compression, Prometheus is dead code (`SetupPrometheus` in `internal/platform/server/metrics/prometheus.go` is never called, no `/metrics` route, no counters), and there is no `/debug/pprof` for production profiling.

2. **Architecture seams.** `internal/modules/auth` is a god-dependency: 9+ modules import it, and 142 call sites across 27 files read request context via `auth.GetRequestContext`/`GetIdentity`/etc. This couples every module to the whole auth graph (Stytch adapters included). Separately, webhook handlers bypass the domain layer: `billing/handler.go` imports `infra/polar` + `infra/mercadopago` directly for signature verification, and `invoicing/handler.go` imports `infra/siigo` (including `siigo.Config`), violating Clean Architecture direction (handler -> app -> domain).

## What Changes

- **JWKS TTL compliance:** `jwksCacheTTL` reduced from 24h to 5 minutes to match the `TTL <= 300s` invariant.
- **Response compression:** new gzip middleware for compressible backend responses; SSE chat stream excluded.
- **Prometheus wiring:** `/metrics` endpoint registered; request counter + latency histogram middleware added.
- **Profiling:** `/debug/pprof` registered, gated behind a config flag (off in prod by default).
- **Auth context seam:** new `internal/platform/authcontext` package owns `Identity`, `RequestContext`, and the Gin/context accessors; 27 files migrate from `auth.*` reads to `authcontext.*`; `auth` middleware writes via the seam. Non-breaking: no behavior or signature change for consumers beyond the import.
- **Webhook verifier seams:** `billing/domain.WebhookVerifier` and `invoicing/domain.WebhookVerifier` interfaces; infra adapters implement them; handlers depend on the interfaces instead of importing infra packages.

## Capabilities

### New Capabilities
- `auth-context-seam`: platform-owned request identity/org context accessors that modules read without importing the auth module.

### Modified Capabilities
- `stytch-authorization`: JWKS local cache TTL contract is tightened to be explicit (TTL <= 300s) so the invariant is testable in spec form.
- `production-health-and-ops`: new requirements for response compression, a Prometheus `/metrics` endpoint, and gated profiling.

## Impact

- **Backend (`go-b2b-starter/internal/`):** new `platform/authcontext` package; new `platform/server/middleware/compression.go` + metrics middleware; edits to `jwks_cache.go`, `billing/handler.go`, `invoicing/handler.go`, 27 files touching `auth.*` context reads, `bootstrap/init_mods.go` (registration), `api/provider.go` (metrics wiring).
- **Config:** new `PPROF_ENABLED` env (default false).
- **Dependencies:** none new — gzip uses stdlib `compress/gzip`.
- **Stytch:** no tenant policy changes; JWKS TTL is purely local key freshness. Per constitution, cached-JWKS read-only fallback is unchanged.
