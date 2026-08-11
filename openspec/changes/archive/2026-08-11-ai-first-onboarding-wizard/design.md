## Context

Signup is a two-step form (`use-signup-flow.ts`, steps `account` | `organization`). Verification lands on `/authenticate` which redirects to `/dashboard`. `DashboardHome` renders KPI cards and quick actions — no first-run guidance. The AI stack already exists: `whatsapp-agent` copilot, metered AI credits, knowledge RAG, agent settings/guardrails. Nothing introduces the user to it.

## Goals / Non-Goals

**Goals:**
- Guided signup wizard that captures business context and sets AI expectations.
- First-run checklist on the dashboard that reflects real completion state.
- Assistant introduction surfaced to new users.

**Non-Goals:**
- No backend/DB persistence of new context fields (assumption above).
- No auth mechanism changes; Stytch contract untouched.
- No plan-purchase forced gate (plan choice remains a nudge, handled by `modern-plan-pricing-ux`).

## Decisions

1. **Extend the existing wizard, don't replace it.** `use-signup-flow.ts` gains a `business` step between `organization` and submit. Step machine stays linear; `canContinue*` guards extended. The Stytch bootstrap call is unchanged.
2. **First-run state derived, not stored.** The dashboard checklist computes completion from real data: WhatsApp connected (`useWhatsAppConfigQuery`), subscription active (`subscriptionState`), first conversation opened (inbox visited / conversations present), assistant introduced (dismiss flag). No new backend endpoint.
3. **Assistant intro = lightweight surface.** A dismissible "Meet your assistant" panel on the dashboard for new orgs explaining the copilot (drafts replies, human approves) and knowledge base, linking to Settings → agent. Reuses existing agent model/APIs; no new API.
4. **Checklist component `components/onboarding/first-run-checklist.tsx`** renders steps with status (done/todo) and links; `dashboard-home.tsx` mounts it when completion < 100%.
5. **Client-side context capture** stores WhatsApp-readiness + goal in localStorage and prefills checklist priorities (e.g., "connect WhatsApp" first when readiness is "already have WhatsApp"). Marked as assumption; no schema change.
6. **Spanish-first** copy via `lib/copy` namespace `onboarding` from `standardize-spanish-first-copy`.

## Risks / Trade-offs

- **Derived checklist can misfire** if a query fails to load; mitigation: treat unknown state as "not done" and show a retry/loading state.
- **localStorage context may be lost** (cleared browser, different device); mitigation: it only shapes priority, not gate behavior — the checklist is still functional.
- **Scope creep toward a paid gate:** explicitly a nudge; plan purchase belongs to the pricing change.
- **Dependency risk:** blocked until the copy layer from `standardize-spanish-first-copy` lands; coordinate merge order.
