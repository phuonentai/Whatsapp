## Context

The `ApiClient` (api-client.ts:25-79) exposes `get`/`post`/`put`/`delete`; `RequestOptions` (line 20) has only `headers`/`skipAuth`. The CRM, conversation, and WhatsApp-config repositories were written against a richer contract: `crm-repository.ts` passes `{ params }` on 6 GET calls, and `conversation-repository.ts` / `whatsapp-config-repository.ts` call `apiClient.patch`. `query-keys.ts` defines 32 namespaces but no `whatsappConfig`. `plans-modal.tsx:104-111` narrows `ActionResult` incorrectly (TS2339 in the `else` branch).

Environment: Next.js 16 + React 19, TypeScript strict, TanStack Query v5, sonner toasts mounted, node v24.

## Goals / Non-Goals

**Goals:**
- Zero `tsc` errors (12-error baseline in 7 files, verified).
- Add `params` to `RequestOptions` with correct query-string serialization (skip `undefined`, URL-encode, `?`/`&` joining).
- Add `ApiClient.patch<T>` mirroring `put`.
- Add `queryKeys.whatsappConfig` (`all`, `detail()`).
- Fix `plans-modal.tsx` narrowing with zero behavior change.

**Non-Goals:**
- Converting existing manual `URLSearchParams` call sites (conversation repo) to the new `params` option.
- Any backend, DB, auth, or behavioral change.

## Decisions

### D1: `params` serialization in `request()`

Add `params?: Record<string, string | number | undefined>` to `RequestOptions`. In `request()` (line 89), after building the base URL, serialize non-undefined entries with `URLSearchParams` and append to the endpoint: `?` if the endpoint has no query string, `&` if it already does. The verb methods (`get`/`post`/`put`/`patch`/`delete`) forward `options` through `applyOptions`; add `params` to the returned object so `request()` receives it.

**Alternatives considered:** per-verb serialization in `get` only (rejected — future verbs need it); a separate `buildQueryString` exported helper (rejected — internal detail).

### D2: `patch<T>` mirrors `put`

```ts
async patch<T>(endpoint: string, body?: any, options?: RequestOptions): Promise<T> {
  return this.request<T>(endpoint, {
    method: "PATCH",
    ...this.applyOptions(body, options),
  });
}
```

Body optional — `toggleConfig` calls it without a body.

### D3: `whatsappConfig` query keys

```ts
whatsappConfig: {
  all: ["whatsappConfig"] as const,
  detail: () => [...queryKeys.whatsappConfig.all, "detail"] as const,
},
```

Matches `use-whatsapp-config-query.ts:7` (`detail()`) and `use-upsert-whatsapp-config.ts:14` (`all`).

### D4: `plans-modal.tsx` narrowing

Restructure the checkout branch to check `!result.success` first (narrows to the error variant), then handle success with/without `checkoutUrl`. Behavior identical: error toast + reset on failure; redirect only when `checkoutUrl` present.

## Risks / Trade-offs

- **Query-string edge cases**: `searchContacts` passes `{ q, ...params }`; `undefined` values must not produce `key=undefined`. Mitigated by skip-undefined serialization; covered by a unit test.
- **Endpoint already carrying `?`**: conversation repo builds its own query strings but never passes `options.params`, so no double-encoding today; the `&` branch is defensive.
- **Baseline drift**: `queryKeys.crm` landed from parallel work; only the verified 12 errors are in scope (task 1.1 snapshots the baseline).
