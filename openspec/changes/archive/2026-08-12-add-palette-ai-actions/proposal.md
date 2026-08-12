# Proposal: add-palette-ai-actions

## Why

The ⌘K command palette is navigation-only (`command-registry.ts` — 14 destinations, all sidebar/settings views). In 2026 the palette is the universal AI entry point ("Cmd+K is the new hamburger menu"; Notion/Linear/Slack embed AI commands in it). This product's AI surfaces (knowledge assistant, AI audience builder, context summaries) are reachable only by clicking through menus — the fastest keyboard surface has no AI.

## What Changes

- **Registry**: add an `AiAction` type + `aiActionRegistry` (parallel to `commandRegistry`, so global `g <key>` shortcuts and existing destinations are untouched). Three actions, all Spanish-first:
  1. **Preguntar al asistente** → `/dashboard/knowledge`
  2. **Nueva conversación de IA** → `/dashboard/knowledge` + triggers a real new-chat (store signal consumed by `knowledge-content.tsx`, whose `handleNewChat` is currently local state at :150-153)
  3. **Audiencia IA para campañas** → `/dashboard/crm?view=campanas` (the AI audience builder surface)
- **Palette**: `command-palette.tsx` renders a new "IA" `CommandGroup` above the navigation sections, filtered by the same query; selecting an action executes it and closes the palette.
- **Store**: `command-palette-store.ts` gains a monotonic `aiNewChatSignal` + `requestNewAiChat()`; `knowledge-content.tsx` subscribes and resets to a fresh chat when it increments.
- **Copy**: action titles/keywords under `ui` copy (Spanish-first).
- **Frontend only — zero backend changes.**

## Capabilities

### New Capabilities

- `command-palette-ai-actions`: AI entry points in the ⌘K palette — knowledge assistant, new AI chat (real chat reset via a store signal), and the AI campaign audience builder — as a dedicated IA group above navigation.

### Modified Capabilities

None.

## Impact

- **Code**: `next_b2b_starter/` — `lib/command-registry.ts` (AiAction + registry), `components/common/command-palette.tsx` (IA group + execution), `lib/stores/command-palette-store.ts` (signal), `app/dashboard/knowledge/components/knowledge-content.tsx` (signal subscription), `lib/copy/ui.ts`, palette tests + knowledge bridge test.
- **Dependencies**: none new.
- **Systems**: none — pure frontend UX surface over existing routes.

## Non-Goals

- No change to navigation destinations, `g <key>` shortcuts, or search mode.
- No new AI capability — actions point at existing AI surfaces (knowledge chat, audience builder); the new-chat reset reuses the existing `handleNewChat`.
- No backend, schema, permission, or Stytch changes; no local credential storage.

## Rollback

- **Git state**: revert the touched files (`command-registry.ts`, `command-palette.tsx`, `command-palette-store.ts`, `knowledge-content.tsx`, `lib/copy/ui.ts`, tests, this change's artifacts). All additions are additive; no migration, no data.
- **Stytch tenant policy state**: no policy changes, so no policy rollback required.
