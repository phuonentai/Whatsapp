## Why

The Next.js frontend does not compile: 12 TypeScript errors across 7 files in the API-client, WhatsApp/inbox hook, and billing layers. `ApiClient` lacks a `patch` method and `RequestOptions` lacks `params` (both required by the conversation, WhatsApp-config, and CRM repositories), the `queryKeys.whatsappConfig` namespace is missing while three hooks depend on it, and `plans-modal.tsx` has a narrowing bug on `ActionResult`. Every CRM write-side wave (`crm-write-side-ui` and beyond) sits on a green build; this change restores it.

## What Changes

- Add `params` to `RequestOptions` and serialize it into the request URL query string (skip `undefined`, safe `?`/`&` joining, URL-encoded via `URLSearchParams`).
- Add `ApiClient.patch<T>(endpoint, body?, options?)` mirroring the existing `put` with method `PATCH`.
- Add the `queryKeys.whatsappConfig` namespace (`all`, `detail`) matching existing hook call sites.
- Fix `plans-modal.tsx` checkout-result narrowing without changing behavior.
- Verify zero `tsc` errors, `pnpm lint`, and `pnpm build` green.

## Capabilities

### New Capabilities

- `frontend-api-client`: the frontend API client contract — PATCH support, query-parameter serialization, and query-key namespaces that match hook usage.

### Modified Capabilities

- none (pure build repair; no behavioral requirement changes)

## Impact

- **FE**: `lib/api/api/client/api-client.ts`, `lib/hooks/queries/query-keys.ts`, `components/billing/plans-modal.tsx`. No repository logic changes.
- **BE**: none. **DB**: none. **Auth boundary**: no change — no Stytch contracts, credentials, sessions, or RBAC touched.
- **Coordination**: `queryKeys.crm` was already added by in-flight work (uncommitted); this change does not re-add it and documents that `crm-write-side-ui` task 1.1/1.2 becomes satisfiable once this lands.
- **Rollback**: revert the commits (pure additive code); Stytch state requires no rollback (never mutated).

## Non-Goals

- The 39-error baseline is not the target; only the 12 current errors are fixed.
- No feature work, no behavior changes, no spec/requirement changes to existing capabilities.
- Local storage of credentials, MFA tokens, or session tokens (forbidden by constitution; unchanged).
