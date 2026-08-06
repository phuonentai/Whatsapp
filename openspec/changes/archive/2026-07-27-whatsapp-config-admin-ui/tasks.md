## 1. Backend: ConfigService

- [x] 1.1 Create `ConfigService` interface and implementation in `internal/modules/whatsapp/app/services/config_service.go` with methods: `GetConfig(ctx, orgID) -> *WhatsAppConfig`, `UpsertConfig(ctx, orgID, input) -> *WhatsAppConfig`, `ToggleConfig(ctx, orgID) -> *WhatsAppConfig`
- [x] 1.2 Implement `GetConfig` — calls `ConfigRepository.GetByOrganizationID`, masks `WebhookSecret` and `VerifyToken` (first 6 + last 4 chars), returns result or `ErrConfigNotFound`
- [x] 1.3 Implement `UpsertConfig` — if config exists, update with partial merge (preserve secrets if empty or masked); if not, create new. Validate required fields for new configs. Handle UNIQUE constraint violations
- [x] 1.4 Implement `ToggleConfig` — reads existing config, flips `IsActive`, calls `Update`
- [x] 1.5 Implement `maskSecret(s string) string` helper: returns `"wh***abcd"` format for values >= 10 chars, `"****"` for shorter values
- [x] 1.6 Implement `isMasked(s string) bool` helper: detects masked values (contains `***` or is exactly `****`)

## 2. Backend: Management handler

- [x] 2.1 Add `configService` field to existing `Handler` struct, update `NewHandler` constructor to accept `ConfigService`
- [x] 2.2 Implement `HandleGetConfig(*gin.Context)` — extracts `organization_id` from context, calls `configService.GetConfig`, returns masked config or 404
- [x] 2.3 Implement `HandleUpsertConfig(*gin.Context)` — parses JSON body, calls `configService.UpsertConfig`, returns masked config or appropriate error (400 validation, 409 conflict)
- [x] 2.4 Implement `HandleToggleConfig(*gin.Context)` — extracts `organization_id` from context, calls `configService.ToggleConfig`, returns masked config or 404
- [x] 2.5 Update provider (`provider.go`) to inject `ConfigService` into `Handler` via DI

## 3. Backend: Routes and verify_token fix

- [x] 3.1 Modify `routes.go` to register two route groups: existing public webhook group (unchanged, no auth) and new management group under `/api/v1/whatsapp` with `auth`, `org_context`, and `subscription` middleware
- [x] 3.2 Register `GET /config` with `auth.RequirePermissionFunc("org", "manage")` pointing to `HandleGetConfig`
- [x] 3.3 Register `PUT /config` with `auth.RequirePermissionFunc("org", "manage")` pointing to `HandleUpsertConfig`
- [x] 3.4 Register `PATCH /config/toggle` with `auth.RequirePermissionFunc("org", "manage")` pointing to `HandleToggleConfig`
- [x] 3.5 Fix `VerifyChallenge` in `webhook_service.go` to iterate active configs and compare `hub.verify_token` against each stored `verify_token`; return 403 if no match

## 4. Backend: API provider wiring

- [x] 4.1 Add `WhatsAppRoutes *whatsapp.Routes` to the `moduleRoutes` struct in `internal/api/provider.go`
- [x] 4.2 Include `whatsAppRoutes` in the `registerAPI` provider function's constructor
- [x] 4.3 Add `srv.RegisterRoutes(modules.WhatsAppRoutes.Routes, server.ApiPrefix)` in the `Invoke` block
- [x] 4.4 Add `whatsapp.NewProvider(container).RegisterDependencies()` call in `setupDependencies` (already in `init_mods.go` via `whatsapp.Init`)

## 5. Frontend: API client and types

- [x] 5.1 Create `lib/api/api/dto/whatsapp-config.dto.ts` with `WhatsAppConfigDto` interface matching the Go `WhatsAppConfig` struct fields (id, organization_id, phone_number_id, business_phone, webhook_secret, verify_token, app_id, is_active, created_at, updated_at)
- [x] 5.2 Create `lib/models/whatsapp-config.model.ts` with `WhatsAppConfig` frontend model interface and `WhatsAppConfigInput` for form submission
- [x] 5.3 Create `lib/api/api/repositories/whatsapp-config-repository.ts` with `WhatsAppConfigRepository` class: `getConfig()`, `upsertConfig(input)`, `toggleConfig()`
- [x] 5.4 Add `"whatsapp-config"` query key family to `lib/hooks/queries/query-keys.ts`

## 6. Frontend: TanStack Query hooks

- [x] 6.1 Create `lib/hooks/queries/use-whatsapp-config-query.ts` — `useQuery` wrapping `whatsappConfigRepository.getConfig()`, enabled by default when user has `org:manage` permission
- [x] 6.2 Create `lib/hooks/mutations/use-upsert-whatsapp-config.ts` — `useMutation` wrapping `whatsappConfigRepository.upsertConfig()`, invalidates `whatsapp-config` query key on success, shows success/error toast
- [x] 6.3 Create `lib/hooks/mutations/use-toggle-whatsapp-config.ts` — `useMutation` wrapping `whatsappConfigRepository.toggleConfig()`, invalidates `whatsapp-config` query key on success

## 7. Frontend: WhatsApp config section component

- [x] 7.1 Create `app/dashboard/settings/components/whatsapp-config-section.tsx` as a client component
- [x] 7.2 Implement empty state: "No WhatsApp number connected" with guidance text and an empty form ready for input
- [x] 7.3 Implement form with fields: `phone_number_id` (text), `business_phone` (text), `app_id` (text), `webhook_secret` (password, with helper "Leave blank to keep current"), `verify_token` (password, same helper)
- [x] 7.4 Implement masked display: when config exists, pre-fill webhook_secret/verify_token with `"••••••••"` placeholder
- [x] 7.5 Add "Save" button that calls `upsertConfig` mutation
- [x] 7.6 Add `is_active` toggle (Switch component) that calls `toggleConfig` mutation with confirmation toast
- [x] 7.7 Add loading state: skeleton placeholder while config query is loading
- [x] 7.8 Add error state: error message with "Retry" button when fetch fails (non-404)
- [x] 7.9 Add inline validation: show field-level errors on save failure

## 8. Frontend: Settings page integration

- [x] 8.1 Add `"whatsapp"` to the `SettingsView` union type literal in `settings-content.tsx`
- [x] 8.2 Add `whatsapp` entry to `DETAIL_META` with title "Messaging" and description "Connect your WhatsApp Business account to receive and manage messages."
- [x] 8.3 Add `whatsapp` entry to `parseViewParam` validation list
- [x] 8.4 Add WhatsApp card to `overviewSections`: key `"whatsapp"`, icon `MessageCircle` (or similar Lucide icon), value derived from config query (phone number or "Not connected"), gated by `canManageMembers`
- [x] 8.5 Add `case "whatsapp"` in `renderDetailContent()` returning `<WhatsAppConfigSection />`
- [x] 8.6 Add permission guard: if user navigates to `?view=whatsapp` without `org:manage`, redirect to overview (reuse existing `useEffect` pattern for members/subscription)

## 9. Verification

- [x] 9.1 Add `GetWhatsAppConfigByVerifyToken` query to SQLC and generated Go code manually (make sqlc requires Docker + network, which is unavailable in this environment)
- [ ] 9.2 Run `make build` to verify the Go backend compiles (requires Go toolchain — blocked by network issues downloading modules in this environment; code follows existing patterns)
- [ ] 9.3 Run `make test` to verify all tests pass (same dependency issue as build)
- [ ] 9.4 Run `pnpm lint` in `next_b2b_starter/` to verify frontend linting (requires Next.js CLI configured — failing due to project root detection, not code)
- [x] 9.5 Run `pnpm build` in `next_b2b_starter/` to verify frontend compiles ✓
