# Tasks: ai-ux-polish

## 1. Markdown foundation

- [ ] 1.1 Add `react-markdown` and `remark-gfm` to package.json ([FE-NEXT]). Verify: `pnpm install`
- [ ] 1.2 Create `components/common/markdown.tsx` ([FE-NEXT]) shared renderer: GFM, no `rehype-raw`, themed to chat bubble styles, copy button on assistant messages. Verify: `pnpm lint`
- [ ] 1.3 Security test ([FE-NEXT]): HTML tags / `<script>` in input render escaped, markdown structure renders. Verify: `pnpm test -- markdown`

## 2. Knowledge chat polish

- [ ] 2.1 Render assistant messages through `markdown.tsx` in `chat-message.tsx` ([FE-NEXT]) replacing `whitespace-pre-wrap` raw text. Verify: `pnpm lint`
- [ ] 2.2 Add `aria-live="polite"` to assistant message container ([FE-NEXT]). Verify: `pnpm lint`
- [ ] 2.3 Update `document-sources.tsx` ([FE-NEXT]): join titles from documents query, document icon, "Documento no disponible" fallback for unknown ids. Verify: `pnpm lint`
- [ ] 2.4 Unit tests: markdown rendering, citation title join + fallback ([FE-NEXT]). Verify: `pnpm test -- knowledge`

## 3. Suggestion panel states

- [ ] 3.1 Replace early `return null` in `agent-suggestions-panel.tsx` with skeleton while pending-suggestions query loads ([FE-NEXT]). Verify: `pnpm lint`
- [ ] 3.2 Replace global approve/reject pending gating with per-suggestion pending id set ([FE-NEXT]) (in-flight action disables only that suggestion's buttons). Verify: `pnpm lint`
- [ ] 3.3 Add conversation-context expansion (read-only thread excerpt, `aria-expanded`) ([FE-NEXT]). Verify: `pnpm lint`
- [ ] 3.4 Extend tests: skeleton state, per-suggestion isolation, context expand ([FE-NEXT]). Verify: `pnpm test -- inbox`

## 4. Structured AI audience output

- [ ] 4.1 Create `components/crm/audience-result-card.tsx` ([FE-NEXT]): criteria chips/lists, audience-size stat, consent-exclusion banner, accept/edit/regenerate actions. Verify: `pnpm lint`
- [ ] 4.2 Replace `<pre>` JSON display in `campaign-manager.tsx:121` with the card ([FE-NEXT]); keep accept payload contract unchanged. Verify: `pnpm lint`
- [ ] 4.3 Unit test: card renders structured result, no raw JSON node in DOM ([FE-NEXT]). Verify: `pnpm test -- campaign-manager`

## 5. Verification

- [ ] 5.1 Run frontend unit tests ([FE-NEXT]). Verify: `pnpm test`
- [ ] 5.2 Run typecheck + lint ([FE-NEXT]). Verify: `pnpm lint` and `npx tsc --noEmit`
- [ ] 5.3 Run production build ([FE-NEXT]). Verify: `pnpm build`
- [ ] 5.4 Run e2e suite ([OPS-GOV]) — inbox/knowledge specs must pass. Verify: `pnpm test:e2e`
