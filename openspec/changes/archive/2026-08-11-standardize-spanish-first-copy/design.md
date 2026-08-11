## Context

`next_b2b_starter` has no i18n layer. Copy is hardcoded inline in JSX across ~10 files. Strings mix English and Spanish depending on author and age of the surface. Developer jargon (Meta tokens, API concepts) appears in primary UI. Target users are Colombian SMEs.

## Goals / Non-Goals

**Goals:**
- One typed copy namespace all UI copy flows through: `lib/copy/`.
- Spanish-first default strings, English fallback constant for untranslated keys.
- Developer/Meta tokens confined to the collapsed Advanced settings panel.
- Committed tone & voice guide.

**Non-Goals:**
- No i18n framework, no locale routing, no language switcher.
- No backend changes; no copy API.
- No translation tooling or external localization pipeline.

## Decisions

1. **Typed copy layer, not a framework.** `lib/copy/ui.ts` exports a `ui` object (or `t()` function with literal keys) grouped by surface: `auth`, `billing`, `whatsapp`, `inbox`, `dashboard`, `agent`, `common`. TypeScript literal types give compile-time key safety. Rationale: app is single-locale (Spanish-first); a full i18n stack adds routing/plumbing with no user-facing benefit today.
2. **Spanish-first contract.** Each key stores the Spanish string; an English fallback exists only where a migration is incomplete, so components never render an empty label.
3. **Jargon confinement.** Meta developer concepts remain but only inside the collapsed "Advanced settings" panel in `whatsapp-config-section.tsx`; primary copy explains what the action does ("Connect your business WhatsApp to receive and manage messages here"), not the mechanism.
4. **Rewrite connect micro-steps** (`MICRO_STATUS_STEPS`) to user language, e.g. "Conectando tu WhatsApp…", "Validando la conexión…", "Todo listo".
5. **Tone & voice guide** committed at `next_b2b_starter/docs/ui-copy.md`: Spanish-first, tú form, benefit-first, short sentences, no internal jargon, error copy states what happened + what to do next.
6. **Component tests updated** to assert copy-layer lookup or final Spanish strings rather than deleted inline strings.

## Risks / Trade-offs

- **Churn risk:** every touched component has a test asserting old strings; the change must update tests in the same commit (verification gate).
- **Scope creep:** without a hard key list, engineers may drift; mitigation is the committed tone guide + code-review convention to route copy through `lib/copy/`.
- **English fallback may mask untranslated keys** and silently ship English; mitigation is a lint-style check (or manual sweep in tasks) ensuring no key resolves to fallback in the primary flows.
