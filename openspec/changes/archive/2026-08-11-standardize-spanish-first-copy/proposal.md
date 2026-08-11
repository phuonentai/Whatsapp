## Why

The UI speaks in two languages at once: English shell strings ("Choose your plan", "Full Name", "No messages yet") sit beside Spanish CRM and billing copy ("Suscripción activa", "Actualiza tu plan", "Error al cargar"), and developer jargon leaks into user-facing surfaces ("Webhook Secret", "WABA ID", "E.164 format", "Coexistence session"). The target market is Colombia (PSE/Nequi, Ley 1581, CC/NIT). A split-language, jargon-heavy interface reads as unfinished and hurts trust precisely at the moments that matter: signup, payment, and WhatsApp connect.

## What Changes

- Introduce a typed, centralized copy layer `lib/copy/` with Spanish-first strings and English fallback, eliminating hardcoded inline copy across the onboarding, billing, WhatsApp-config, inbox, dashboard, and agent-settings surfaces.
- Standardize voice: plain, human, benefit-first; Meta/developer tokens (webhook secret, verify token, WABA ID, access token, API version) appear only inside the collapsed "Advanced settings" panel, not in primary user-facing copy.
- Rewrite connect micro-status steps from developer language ("Verifying Coexistence session...", "Establishing secure token & webhooks...") to user language.
- Add a short tone & voice guide (committed under `docs/`) codifying the language policy for future copy.
- No URL routing, locale detection, or user-facing language switcher in this change.

## Capabilities

### New Capabilities
- `ui-copy`: the typed copy layer (`lib/copy/`), its key namespace, the Spanish-first/English-fallback contract, and the tone & voice policy that all UI copy must follow.

### Modified Capabilities
- `billing-provider-ux`: billing surfaces render copy through the copy layer in Spanish-first voice instead of mixed-language hardcoded strings.
- `settings-ui`: settings sections (subscription, WhatsApp config, agent) render copy through the copy layer with developer tokens confined to advanced panels.
- `inbox-ui`: inbox empty/failure states render Spanish-first copy through the copy layer.
- `ui-error-recovery`: error and mutation-failure copy is sourced from the copy layer in Spanish-first voice.

## Impact

- Frontend: `next_b2b_starter/lib/copy/` (new), plus string replacement in `app/signup/page.tsx`, `components/billing/plans-modal.tsx`, `components/billing/subscription-paywall.tsx`, `app/dashboard/settings/components/whatsapp-config-section.tsx`, `app/dashboard/settings/components/subscription-tab.tsx`, `app/dashboard/settings/components/agent-settings-section.tsx`, `app/dashboard/components/dashboard-home.tsx`, inbox empty-state components, and common error/empty-state components.
- Backend: none. This change is frontend-only; no API, DB, or Stytch contract changes.
- Tests: component tests asserting the previous hardcoded strings must be updated to assert copy-layer keys or the Spanish strings.
- Non-Goals: no i18n framework, no locale routing, no language switcher, no backend copy endpoints. No authentication flow or data-persistence change; Stytch B2B remains the sole authority for identity and RBAC (see Stytch B2B API contracts — https://stytch.com/docs/api-reference/b2b/api/overview).
- Rollback: revert the copy layer commit in Git; no Stytch tenant state is modified by this change, so no Stytch rollback applies.
