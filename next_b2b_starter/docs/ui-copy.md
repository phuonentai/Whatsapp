# UI Copy Guide — Spanish-first

Tone and voice policy for all user-facing copy in the app. Target users are Colombian SMEs; the primary market language is Spanish (Colombia).

## Language policy

- **Spanish-first.** Every user-facing string resolves from the typed copy layer at `lib/copy/ui.ts` (namespace `ui`). Never hardcode a user-facing string inline in JSX.
- **English fallback** exists in `lib/copy/ui.ts` (the `en` mirror) only where a migration is incomplete, so a label never renders empty. Primary flows (signup, plans modal, WhatsApp connect) must always resolve to Spanish.
- No i18n framework, no locale routing, no language switcher. This is a single-locale Spanish-first product.

## Voice

- **Tú form.** Address the user directly and simply ("Conecta tu WhatsApp", not "Conectar WhatsApp" imperative-isolated; prefer warm direct tú where natural).
- **Benefit-first.** Lead with what the action does for the user, not the mechanism.
  - Bad: "Establishing secure token & webhooks…"
  - Good: "Conectando tu WhatsApp…"
- **Plain language.** No internal jargon in primary copy. Developer/Meta terms (webhook secret, verify token, WABA ID, permanent access token, API version, Graph API URL, Embedded Signup) appear **only** inside collapsed advanced panels.
- **Short sentences.** One idea per sentence. Colombian Spanish register (tú/vos), plain vocabulary.

## Error copy pattern

Every error or mutation failure states **what happened + what to do next**:

1. What happened (short, human): "No se pudieron cargar los planes."
2. Next step: "Inténtalo de nuevo."
3. Optional retry action labeled "Reintentar".

## Consistency rules

- Route all copy through `lib/copy/ui.ts`; unknown keys fail the build, so the type system guards copy.
- When adding copy: pick the right namespace (`auth`, `billing`, `whatsapp`, `inbox`, `dashboard`, `agent`, `common`), keep the Spanish string, mirror it in `en`.
- Keep user-facing status labels in Spanish ("Activo", "Conectado", "En pausa").
