## Why

Onboarding today is a bare two-step form (account, org) followed by a dead-end magic-link redirect into a generic dashboard. There is no business-context capture, no plan guidance, no first-run guidance, and no moment where the AI assistant is introduced. Modern SaaS (2026) expectations are AI-first onboarding: capture the business goal, set expectations for the assistant, and walk the user into value immediately after verification.

## What Changes

- Rework the signup flow into a guided, AI-first wizard: account → organization → business context (vertical, WhatsApp readiness, goal) → confirmation with a plan nudge.
- Introduce a first-run onboarding checklist surface on the dashboard for new organizations: connect WhatsApp, choose a plan, meet your AI assistant, explore the inbox. Checklist items complete as the user takes real actions.
- Add an AI assistant introduction moment ("your assistant will learn your business as chats arrive") tied to the existing `whatsapp-agent` copilot and knowledge base.
- Keep the Stytch B2B bootstrap contract unchanged (native invite via `Members.Create` with `SendInvite: true`, no `owner_password`, structured error codes). This change is frontend-first; the existing signup endpoints are reused.
- Depends on `standardize-spanish-first-copy` for the typed copy layer (Spanish-first).

## Capabilities

### New Capabilities
- `ai-onboarding`: the guided AI-first signup wizard, the first-run dashboard checklist, and the assistant introduction moment.

### Modified Capabilities
- (none) — the wizard reuses the existing signup/bootstrap contract in `signup-stytch-compliance`; no requirement there changes.

## Impact

- Frontend: `app/signup/page.tsx`, `hooks/use-signup-flow.ts`, `app/authenticate/page.tsx` (post-verify landing), `app/dashboard/components/dashboard-home.tsx` (checklist surface), new wizard and checklist components under `components/onboarding/`.
- Backend: none required for the wizard; existing signup endpoints are reused. Business-context fields beyond `industry` (WhatsApp readiness, goal) are captured client-side and surfaced in the checklist — see Assumptions.
- Dependencies: typed copy layer from `standardize-spanish-first-copy`.
- Authentication flow: signup UX changes but the Stytch B2B contract is unchanged — organization bootstrap continues to use Stytch `Members.Create` with `SendInvite: true`, and the payload continues to exclude `owner_password` (https://stytch.com/docs/api-reference/b2b/api/overview).
- Rollback: revert the wizard commit in Git; no Stytch tenant policy state is created or modified beyond the standard org bootstrap, so no separate Stytch rollback applies.
- Non-Goals: no new auth mechanism, no password/local credential storage (passwords remain forbidden by `signup-stytch-compliance`), no backend persistence of goal/readiness fields in this change, no changes to the Stytch RBAC model.

## Assumptions

- Business-context fields beyond the existing `industry` (WhatsApp readiness, business goal) have no persisted backend column today; they will be captured client-side and used to shape the checklist until a persistence change is proposed.
- The post-verification landing currently redirects straight to `/dashboard` (`DEFAULT_DESTINATION`); the checklist will mount on the dashboard home for organizations that have not completed first-run steps.
