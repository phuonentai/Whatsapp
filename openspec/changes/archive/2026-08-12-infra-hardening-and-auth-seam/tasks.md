## 1. Compliance Fix: JWKS Cache TTL

- [x] 1.1 [BE-INFRA] Change `jwksCacheTTL` in `internal/modules/auth/adapters/stytch/jwks_cache.go:20` from `24 * time.Hour` to `5 * time.Minute`. Verification: `make build` passes; grep shows `jwksCacheTTL = 5 * time.Minute`.

## 2. Response Compression

- [x] 2.1 [BE-INFRA] Create `internal/platform/server/middleware/compression.go`: stdlib gzip middleware that compresses when `Accept-Encoding` includes gzip, content-type is compressible (json/text/html), path not excluded, and no existing `Content-Encoding`. Excluded paths: `/api/example_cognitive/chat` (SSE), `/metrics`. Verification: `make build` passes.
- [x] 2.2 [BE-INFRA] Register compression middleware in `setupMiddleware()` (`internal/platform/server/domain/middleware.go:16`) after RequestID, before request logging. Verification: `make build`; `curl -H "Accept-Encoding: gzip" /api/<json-endpoint>` returns `Content-Encoding: gzip`; SSE chat request returns identity encoding and streams tokens.

## 3. Prometheus Wiring

- [x] 3.1 [BE-INFRA] Call `metrics.SetupPrometheus(engine)` at boot in `internal/api/provider.go` after route registration so `/metrics` is served. Verification: `make build`; `curl /metrics` returns HTTP 200 Prometheus text format.
- [x] 3.2 [BE-INFRA] Add request metrics middleware registering `http_requests_total{method,status,path}` counter and `http_request_duration_seconds{method,path}` histogram; skip `/metrics` and the SSE chat path; register in the middleware chain. Verification: `make build`; hitting a JSON endpoint increments `http_requests_total`.

## 4. Profiling Endpoint

- [x] 4.1 [BE-INFRA] Add `PPROF_ENABLED` config (default false) in `internal/platform/server/config`. Verification: `make build`.
- [x] 4.2 [BE-INFRA] Register `net/http/pprof` under `/debug/pprof` on the Gin engine only when `PPROF_ENABLED` is true (forced off in prod unless explicitly set). Verification: `make build`; with flag unset `/debug/pprof/` is 404; with `PPROF_ENABLED=true` it is 200.

## 5. Auth Context Seam

- [x] 5.1 [BE-DOMAIN] Create `internal/platform/authcontext` package: move `Identity`, `RequestContext` types and the context keys + accessors from `internal/modules/auth/context.go` and `auth.go` (`Set/Get/MustGet Identity`, `Set/Get/MustGet RequestContext`, `GetOrganizationID`, `GetAccountID`, `WithIdentity`, `IdentityFromContext`, `WithRequestContext`, `RequestContextFromContext`). Verification: `make build`.
- [x] 5.2 [BE-DOMAIN] Update `auth` middleware (`internal/modules/auth/middleware.go`) to write identity/request context via `authcontext.SetIdentity` / `authcontext.SetRequestContext`; re-export moved symbols in `auth` for adapter code. Verification: `go test ./internal/modules/auth/...` passes.
- [x] 5.3 [BE-DOMAIN] Migrate CRM, WhatsApp, Instagram files: replace `auth.GetRequestContext`/`MustGetRequestContext`/`GetIdentity`/`GetOrganizationID`/`GetAccountID` reads with `authcontext.*`. Keep `auth` import where `RequirePermissionFunc` is used. Verification: `make build`; `go test ./internal/modules/crm/... ./internal/modules/whatsapp/... ./internal/modules/instagram/...` passes.
- [x] 5.4 [BE-DOMAIN] Migrate remaining files (agent, analytics, billing, campaigns, cognitive, documents, invoicing, organizations, paywall, playbooks, registry, tickets). Verification: `make build`; `go test ./...` passes; `grep -rn "auth.GetRequestContext\|auth.MustGetRequestContext\|auth.GetIdentity\|auth.GetOrganizationID\|auth.GetAccountID" internal | grep -v "/modules/auth/"` returns 0 matches.

## 6. Webhook Verifier Seams

- [x] 6.1 [BE-DOMAIN] Add `billing/domain.WebhookVerifier` interface (`VerifyPolar`, `VerifyMercadoPago`) and move header-name constants into `billing/domain/webhook.go`; implement the interface in `billing/infra/polar` and `billing/infra/mercadopago` adapters wrapping existing `VerifyWebhookSignature`. Existing signature-validation tests must keep passing. Verification: `go test ./internal/modules/billing/...` passes.
- [x] 6.2 [BE-DOMAIN] Provide composite `WebhookVerifier` in billing DI; change `billing.NewHandler` to depend on the interface instead of importing infra packages. Verification: `make build`; `grep "infra/polar\|infra/mercadopago" internal/modules/billing/handler.go` returns nothing.
- [x] 6.3 [BE-DOMAIN] Add `invoicing/domain.WebhookVerifier` interface (`Verify(payload, signature, secret string) error`); implement in `invoicing/infra/siigo` wrapping `VerifyWebhookSignature`. Verification: `go test ./internal/modules/invoicing/...` passes.
- [x] 6.4 [BE-DOMAIN] Change `invoicing.NewHandler` to take verifier + `sandbox bool` + `webhookSecret string` instead of `*siigo.Config`; wire DI. Verification: `make build`; `grep "infra/siigo" internal/modules/invoicing/handler.go` returns nothing.

## 7. Governance

- [x] 7.1 [OPS-GOV] Verify all delta specs present (`specs/stytch-authorization`, `specs/auth-context-seam`, `specs/production-health-and-ops`) and tasks match design. Verification: `openspec status --change "infra-hardening-and-auth-seam"` shows `tasks` done; `openspec validate` passes.

## Verification Gate (post-implementation)

All verification commands run and passed:
- `make build` — passed
- `go build ./...` — passed
- `go test ./...` — passed (all packages)
- `go test ./internal/modules/auth/...` — passed
- `go test ./internal/modules/billing/...` — passed (incl. polar/mercadopago sig tests)
- `go test ./internal/modules/invoicing/...` — passed (incl. siigo sig tests)
- `go test ./internal/modules/crm/... ./internal/modules/whatsapp/... ./internal/modules/instagram/...` — passed
- grep `jwksCacheTTL` shows `5 * time.Minute` — passed
- grep `auth.GetRequestContext|auth.MustGetRequestContext|auth.GetIdentity|auth.GetOrganizationID|auth.GetAccountID` outside `/modules/auth/` returns 0 matches — passed
- grep `infra/polar|infra/mercadopago` in `billing/handler.go` returns 0 — passed
- grep `infra/siigo` in `invoicing/handler.go` returns 0 — passed
- `openspec validate infra-hardening-and-auth-seam` — passed (valid)
- `openspec status` — 16/16 tasks complete

Note: `curl`-based checks (`Content-Encoding: gzip`, `/metrics`, `/debug/pprof` 200/404) require a running server; verified statically via build + code inspection. Dynamic smoke test deferred to deployment.

**Archive deferred:** working tree holds 311 modified files from other in-progress changes (add-client-payments, app-shell-modernization, ai-ux-polish, ui-data-tables). Folding deltas into living specs now would mix unrelated WIP; archive after the working tree is committed/cleaned.
