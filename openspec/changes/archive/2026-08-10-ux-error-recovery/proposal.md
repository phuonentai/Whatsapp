# Change Proposal: ux-error-recovery

## Why

The platform degrades silently on failure. Query errors render as eternal "Cargando..." (ticket-detail.tsx:27, contact-table.tsx:62), inbox send failures produce an unhandled promise rejection with no toast (use-send-message.ts has no onError), and four+ flows use native `window.alert`/`confirm()` (plans-modal.tsx:73, member-list.tsx:227-229, compliance-section.tsx:19-65) that block the page and are inaccessible. Inbound messages have no unread indicators, so agents miss new conversations. This is the P0 cluster of the UI/UX gap analysis.

## What Changes

- Add error + retry UI to every data-loading view: failed queries SHALL render an inline error state with a retry action instead of an infinite loading spinner.
- Add explicit failure UX for inbox message send: toast on error, input text preserved, no unhandled rejection.
- Replace native `window.alert` / `window.confirm` in billing plan-switch, member role change/removal, and compliance forget flows with the existing custom `ConfirmDialog` / sonner-based flows.
- Add unread indicators for conversations with new inbound messages in the inbox conversation list.
- Add `aria-live`/`role="status"` announcements for message arrival and AI-suggestion updates so screen readers report activity.
- Standardize a shared `ErrorState`/retry component used across CRM, tickets, inbox, and settings.

## Capabilities

### New Capabilities
- `ui-error-recovery`: Cross-cutting rules for error/retry rendering, silent-failure prevention, native-dialog ban, and shared error-state component.

### Modified Capabilities
- `inbox-ui`: reply-send failure SHALL toast and preserve the draft; conversation list SHALL show unread indicators for new inbound messages.
- `crm-frontend`: CRM and ticket views SHALL render error/retry states instead of infinite loading.
- `billing-provider-ux`: plan-switch blocking SHALL NOT use `window.alert`.
- `settings-ui`: member role change/removal and compliance forget SHALL NOT use native `window.confirm`.

## Impact

- Frontend only (`next_b2b_starter/`): `app/dashboard/inbox/*`, `components/crm/*`, `components/tickets/*`, `components/billing/*`, `app/dashboard/settings/components/*`, plus a new shared component under `components/common/`.
- No API, schema, or Stytch changes.
- New dependency: none (cmdk NOT included here).

## Non-Goals

- No local credential, password, MFA, or session-token storage — Stytch B2B remains the sole identity/session authority; this change only adds UI feedback around existing API failures.
- No auth-flow changes; no backend route changes.
- No i18n extraction or dark mode (separate changes).

## Rollback

- Git state: revert the change's commits; feature is additive UI, no migrations.
- Stytch tenant policy state: no Stytch resources are created or altered; nothing to roll back.
