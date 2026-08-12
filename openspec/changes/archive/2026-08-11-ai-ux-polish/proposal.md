# Change Proposal: ai-ux-polish

## Why

AI surfaces work but are developer-facing. Knowledge chat renders plain text with no markdown (chat-message.tsx:27,49), citations show "Document #id" instead of titles (document-sources.tsx:39-58), the CRM AI audience builder outputs raw JSON in a `<pre>` (campaign-manager.tsx:121), and the inbox suggestion panel returns `null` while loading (agent-suggestions-panel.tsx:24), popping in with no skeleton. This is the P2 cluster of the UI/UX gap analysis.

## What Changes

- Render AI chat responses as markdown (headings, lists, bold, code) with a safe renderer and copy button per message.
- Show document titles in knowledge chat citation cards (fetch/join titles via existing documents query) with file-type icon and page reference where available.
- Replace the raw JSON `<pre>` AI audience-builder output with a structured spec card UI (segment criteria as labeled chips/lists, estimated audience size, consent-exclusion notice) with accept/edit/regenerate actions.
- Add loading skeletons and per-suggestion pending states to the inbox agent-suggestion panel (no full-panel `null` flash, no global disable of all approve/reject buttons while one is pending).
- Add "view conversation context" affordance for AI suggestions (expanded thread excerpt before approve).
- Add `aria-live="polite"` region so streaming assistant text is announced to screen readers.

## Capabilities

### New Capabilities
- `ai-ux-affordances`: shared AI surface behavior — markdown chat rendering, human-readable citations, structured AI output instead of raw JSON, suggestion loading/state affordances.

### Modified Capabilities
- `knowledge-base-ui`: assistant messages SHALL render markdown; citations SHALL display document titles.
- `inbox-ui`: the agent-suggestion panel SHALL show skeleton loading and per-suggestion pending states (no wholesale disable, no full-panel `null` flash).
- `crm-frontend`: the AI audience builder result SHALL render as a structured UI card, not raw JSON.

## Impact

- Frontend only (`next_b2b_starter/`): `app/dashboard/knowledge/components/*`, `app/dashboard/inbox/components/agent-suggestions-panel.tsx`, `components/crm/campaign-manager.tsx`.
- New dependency: `react-markdown` + `remark-gfm` (or equivalent — decide in design).
- No API, schema, or Stytch changes; document titles already available via the documents query.

## Non-Goals

- No local credential, password, MFA, or session-token storage — Stytch B2B remains the sole identity/session authority.
- No agent pipeline/LLM behavior changes (backend untouched).
- No streaming-protocol changes (SSE already in place).
- No i18n extraction (separate change).

## Rollback

- Git state: revert the change's commits; additive UI, no migrations.
- Stytch tenant policy state: no Stytch resources are created or altered; nothing to roll back.
