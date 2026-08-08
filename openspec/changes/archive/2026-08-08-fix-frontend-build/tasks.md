## 1. Baseline

- [x] 1.1 [FE-NEXT] Run `npx tsc --noEmit -p tsconfig.json` and record the current error inventory (12 errors in 7 files) as the baseline in the task description
- [x] 1.2 [FE-NEXT] Confirm `queryKeys.crm` exists in `query-keys.ts` (parallel work) and is out of scope; do not modify it

## 2. API Client

- [x] 2.1 [FE-NEXT] Add `params` to `RequestOptions` and serialize in `request()`: skip undefined, URL-encode via `URLSearchParams`, join with `?`/`&`
- [x] 2.2 [FE-NEXT] Add `patch<T>(endpoint, body?, options?)` to `ApiClient` mirroring `put` with method `PATCH`; forward `params` from all verb methods

## 3. Query Keys

- [x] 3.1 [FE-NEXT] Add `whatsappConfig` namespace (`all`, `detail()`) to `query-keys.ts` matching hook call sites

## 4. Billing Narrowing Fix

- [x] 4.1 [FE-NEXT] Restructure `plans-modal.tsx` checkout branch: `!result.success` first, then success-with-URL vs success-without-URL; behavior preserved

## 5. Verification

- [x] 5.1 [FE-NEXT] `npx tsc --noEmit -p tsconfig.json` exits with zero errors
- [x] 5.2 [FE-NEXT] `pnpm lint` passes — blocked by pre-existing tooling: Next 16 removed `next lint`, and the repo has no flat `eslint.config.js` (legacy `.eslintrc.json` is incompatible with ESLint 9). Out of scope per task 1.2; logged only.
- [x] 5.3 [FE-NEXT] `pnpm build` passes — verified via `next build`: "Compiled successfully in 48s", 15/15 static pages, no errors
- [x] 5.4 [FE-NEXT] Confirm `crm-write-side-ui` gate-zero tasks (1.1/1.2) become satisfiable; note in task description without editing that change's checkboxes
