# Design: ai-ux-polish

## Context

Frontend-only change in `next_b2b_starter/` polishing the three existing AI surfaces. Verified current state:

- Knowledge chat: SSE streaming works (`use-chat-stream.ts`), optimistic user bubble, sources panel with `Document #id` labels only (`document-sources.tsx:39-58`); messages render `whitespace-pre-wrap` plain text (`chat-message.tsx:27,49`).
- CRM campaign manager: AI audience builder renders result as raw JSON `<pre>` (`campaign-manager.tsx:121`).
- Inbox agent suggestions: panel `return null` while loading (`agent-suggestions-panel.tsx:24`); approve/reject `isPending` disables ALL buttons when any is in flight (`agent-suggestions-panel.tsx:85,103`).
- Documents query exists and returns titles (`use-documents-query.ts`) — citation join is client-side.

Spec contract: specs/ai-ux-affordances/spec.md (new) + deltas for knowledge-base-ui, inbox-ui, crm-frontend.

## Goals / Non-Goals

**Goals**: markdown rendering for AI chat, titled citations, structured AI audience output, skeleton + per-suggestion pending states, conversation-context expansion, live-region announcements, copy button on assistant messages.

**Non-Goals**: no backend/LLM/pipeline changes; no streaming-protocol changes; no new AI capabilities; no i18n.

## Decisions

### D1: `react-markdown` + `remark-gfm` for chat rendering (new dependency)
`react-markdown` is the standard safe-by-default renderer (no raw HTML unless `rehype-raw` explicitly added — we will NOT add it, satisfying the sanitization requirement). `remark-gfm` for tables/strikethrough/task lists.
- Alternatives: `marked`+`DOMPurify` — rejected: sanitization becomes manual; react-markdown is component-based and themable to the existing chat bubble styles.
- Alternatives: hand-rolled renderer — rejected: security-sensitive and incomplete.

### D2: Citation titles joined client-side from the documents query
When rendering source cards, look up titles from the already-available documents query data (keyed by document id); unknown ids render "Documento no disponible" fallback. No new endpoint needed.
- Rationale: spec requires title display + graceful fallback; documents query already returns titles (verified: `use-documents-query.ts`).

### D3: Audience builder result = structured card component
New `components/crm/audience-result-card.tsx`: criteria as chips/lists, audience-size stat, consent-exclusion banner, actions accept/edit/regenerate. The `<pre>` JSON output is replaced; raw payload kept in state for the accept payload, not displayed.
- Alternative: render JSON in styled `<pre>` — rejected: spec explicitly forbids raw JSON as primary presentation.

### D4: Per-suggestion pending state
Replace single `approveMutation.isPending`/`rejectMutation.isPending` boolean gating with per-suggestion pending id set (`pendingSuggestions: Set<string>` state). Skeleton while `usePendingSuggestionsQuery` is loading (replace early `return null`).
- Alternative: mutation-level `variables.id` tracking — acceptable too, but explicit Set state is simpler to test and independent of TanStack internals.

### D5: Context expansion = read-only excerpt panel
Expandable section in the suggestion panel rendering the conversation messages the suggestion was based on (from existing `useMessagesQuery` data), read-only, `aria-expanded` on the toggle.

### D6: Live region for streaming
Assistant message container gets `aria-live="polite"`; token accumulation already batches DOM updates (one update per rendered chunk), so no per-token announcement spam.

## Risks / Trade-offs

- [Markdown rendering introduces XSS surface] → react-markdown default: no raw HTML; no `rehype-raw`; verify with a unit test asserting `<script>`/HTML tags render escaped.
- [react-markdown version drift with React 19] → pin latest v9+ (React 19 compatible); verify via build.
- [Citation join when documents query stale/not fetched] → fallback label covers it; also reuse cache data (staleTime 2min).
- [Per-suggestion pending refactor touches existing approve/reject flows] → keep mutation logic intact; only UI gating changes; existing `agent-suggestions-panel` tests cover approve/reject, extend with pending-isolation test.
- [Audience card must keep the existing accept payload contract] → the create-segment call signature unchanged; card only restyles presentation.

## Migration Plan

1. Add `react-markdown` + `remark-gfm`; extract shared `components/common/markdown.tsx` renderer.
2. Knowledge chat: markdown + copy button + titled citations + live region.
3. Suggestion panel: skeleton, per-item pending, context expansion.
4. Campaign manager: structured audience card.
5. Rollback: git revert per commit; no API/schema/backend changes.

## Open Questions

- Whether markdown renderer should also apply to inbox agent suggestion text (currently plain textarea) — default: suggestions stay plain text (they're pre-approved drafts), markdown only for knowledge assistant messages. Revisit if product wants rich suggestions.
- Copy button confirmation style (toast vs icon swap) — icon swap (check for 2s), matches sonner usage elsewhere.
