## 1. Data layer [FE-NEXT]

- [ ] 1.1 Add `lib/api/` client functions + TanStack Query hooks: `useSiigoStatus` (GET status), `useSiigoConnect`, `useConfirmNumeration`, `useImportPreview`, `useImportConfirm`, `useSync`, `useTestInvoice`, `usePauseInvoicing`, `useResumeInvoicing`. Verify: hooks typed against change-1/2 endpoint responses; `npx tsc --noEmit` on touched files passes (full-file baseline exception documented)
- [ ] 1.2 Add admin hooks: `useAdminOnboardingList`, `useAdminProvision`. Verify: scoped tsc passes; auth client role check included

## 2. Status banner & state map [FE-NEXT]

- [ ] 2.1 Implement `SiigoStatusView` switching on connection state with banners for all states: `none`, `awaiting_setup`, `connected`, `numeracion_ok`, `sandbox_ok`, `paused`, `invoicing_disabled`, `live`. Verify: component tests cover every state; banner never empty
- [ ] 2.2 Render assisted banner "Tu equipo está configurando tu facturación" for `awaiting_setup`; single-line "Facturación desactivada — activa con Siigo" for `invoicing_disabled`. Verify: tests assert exact copy

## 3. Wizard steps [FE-NEXT]

- [ ] 3.1 Step 1 Conectar: credential form (react-hook-form) → connect hook; NIT-mismatch error rendered verbatim. Verify: test — submit calls connect, error inline, no advance on failure
- [ ] 3.2 Step 2 Numeración: render resolución/prefijo/próximo número from GET numeration; confirm button → confirm hook; locked until state `connected`. Verify: test — gating, confirm advances
- [ ] 3.3 Step 3 Importar clientes: preview counts display (nuevos/existentes/duplicados/sin NIT); explicit confirm → confirm hook; result counts + timestamp shown after. Verify: test — no confirm call before user action; result rendered
- [ ] 3.4 Step 4 Prueba en sandbox: test-invoice button + awaiting indicator + success on `sandbox_ok`. Verify: test — button disabled during pending, success state
- [ ] 3.5 Step 5 Activar: enabled at `sandbox_ok`; shows `factura_lista` template approval status (approved/pending from existing WhatsApp config data; pending default if unavailable). Verify: test — both template states

## 4. Kill-switch [FE-NEXT]

- [ ] 4.1 Pause/resume toggle in Siigo section for `live`/`paused` states; status query invalidated after call. Verify: test — toggle calls endpoint, banner updates

## 5. Admin view [FE-NEXT]

- [ ] 5.1 Admin onboarding overview: table (org, state, prefijo, next number, last import run, last error) + nav entry following permission-filtered pattern. Verify: test — non-admin denied, rows render
- [ ] 5.2 Assisted provisioning form inline for `awaiting_setup` rows; success refreshes row, server error verbatim. Verify: test — submit calls provision, error displayed

## 6. Wiring [FE-NEXT]

- [ ] 6.1 Wire `siigo-integration-section.tsx` into `settings-content.tsx` alongside existing sections. Verify: `pnpm build` EXIT 0

## 7. Launch gate [OPS-GOV]

- [ ] 7.1 Run `pnpm lint` (record baseline 9+1, no new errors), component tests (`pnpm test` per repo config), `pnpm build`. Verify: results recorded here; new tests pass
- [ ] 7.2 Record external exceptions: `tsc --noEmit` full-file failure from `lib/auth/stytch/server.ts:178` (owned by another in-flight change) — wizard files scoped-pass. Verify: noted in this tasks.md
- [ ] 7.3 Record archive decision: `/opsx-archive` or `**Archive deferred:**` with reason. Verify: entry present
