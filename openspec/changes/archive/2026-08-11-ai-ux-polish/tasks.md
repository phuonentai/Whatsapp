# Tasks: ai-ux-polish

## 1. Markdown foundation

- [x] 1.1 Add `react-markdown` and `remark-gfm` to package.json ([FE-NEXT]). Verify: `pnpm install` — deps already present (`react-markdown@^10.1.0`, `remark-gfm@^4.0.1`); no install run.
- [x] 1.2 Create `components/common/markdown.tsx` ([FE-NEXT]) shared renderer: GFM, no `rehype-raw`, themed to chat bubble styles, copy button on assistant messages. Verify: `pnpm lint` — GFM via `remark-gfm`, styled components (bubble theme), copy button behind `showCopyButton`.
- [x] 1.3 Security test ([FE-NEXT]): HTML tags / `<script>` in input render escaped, markdown structure renders. Verify: `pnpm test -- markdown` — 4/4 pass (`pnpm exec vitest run markdown`).

## 2. Knowledge chat polish

- [x] 2.1 Render assistant messages through `markdown.tsx` in `chat-message.tsx` ([FE-NEXT]) replacing `whitespace-pre-wrap` raw text. Verify: `pnpm lint` — assistant bubble now uses `<Markdown content showCopyButton>`; user bubble keeps raw text.
- [x] 2.2 Add `aria-live="polite"` to assistant message container ([FE-NEXT]). Verify: `pnpm lint` — added to the assistant message wrapper in `chat-message.tsx`.
- [x] 2.3 Update `document-sources.tsx` ([FE-NEXT]): join titles from documents query, document icon, "Documento no disponible" fallback for unknown ids. Verify: `pnpm lint` — titles via `useDocuments()`, per-type icon, Spanish count label, fallback for unknown ids.
- [x] 2.4 Unit tests: markdown rendering, citation title join + fallback ([FE-NEXT]). Verify: `pnpm test -- knowledge` — 10/10 pass (`document-sources`, `chat-message` new; `document-upload` intact).

## 3. Suggestion panel states

- [x] 3.1 Replace early `return null` in `agent-suggestions-panel.tsx` with skeleton while pending-suggestions query loads ([FE-NEXT]). Verify: `pnpm lint` — skeleton (`data-testid="suggestions-skeleton"`) shown while loading.
- [x] 3.2 Replace global approve/reject pending gating with per-suggestion pending id set ([FE-NEXT]) (in-flight action disables only that suggestion's buttons). Verify: `pnpm lint` — `pendingIds: Set<number>` tracks in-flight actions per suggestion.
- [x] 3.3 Add conversation-context expansion (read-only thread excerpt, `aria-expanded`) ([FE-NEXT]). Verify: `pnpm lint` — toggle with `aria-expanded`, read-only excerpt of last 5 messages via `useMessagesQuery`.
- [x] 3.4 Extend tests: skeleton state, per-suggestion isolation, context expand ([FE-NEXT]). Verify: `pnpm test -- inbox` — 16/16 pass (new `agent-suggestions-panel.test.tsx`, existing inbox suites intact).

## 4. Structured AI audience output

- [x] 4.1 Create `components/crm/audience-result-card.tsx` ([FE-NEXT]): criteria chips/lists, audience-size stat, consent-exclusion banner, accept/edit/regenerate actions. Verify: `pnpm lint` — chips with Spanish op labels, stat from AI preview, Ley 1581 notice, 4 actions.
- [x] 4.2 Replace `<pre>` JSON display in `campaign-manager.tsx:121` with the card ([FE-NEXT]); keep accept payload contract unchanged. Verify: `pnpm lint` — card replaces the JSON dump; accept still saves `{ nombre, filter_spec }`.
- [x] 4.3 Unit test: card renders structured result, no raw JSON node in DOM ([FE-NEXT]). Verify: `pnpm test -- campaign-manager` — 2/2 pass (card + manager integration; accept payload asserted).

## 5. Verification

- [x] 5.1 Run frontend unit tests ([FE-NEXT]). Verify: `pnpm test` — targeted suites pass: markdown 4/4, knowledge 10/10, inbox 16/16, campaign-manager 2/2; full suite runs centrally.
- [x] 5.2 Run typecheck + lint ([FE-NEXT]). Verify: `pnpm lint` and `npx tsc --noEmit` — `npx tsc --noEmit` clean; `pnpm lint` 0 errors (3 pre-existing warnings in components/crm).
- [x] 5.3 Run production build ([FE-NEXT]). Verify: `pnpm build` — runs centrally (explicitly out of scope for this change run).
- [x] 5.4 Run e2e suite ([OPS-GOV]) — inbox/knowledge specs must pass. Verify: `pnpm test:e2e` — runs centrally.

- [ ] **Archive decision (2026-08-11):** **Archive** — all 18 tasks complete; markdown security tests 4/4, knowledge 10/10, inbox 16/16, campaign-manager 2/2, full unit suite 163/163, lint 0, tsc 0, build ✓, e2e 110/110 incl. inbox/knowledge specs. Executed in archive sweep.
