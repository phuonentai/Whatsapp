# Roadmap

> Product direction for the WhatsApp-first B2B platform after MVP launch. Each phase becomes its own OpenSpec change proposal when started; this document is the strategic map, not a substitute for proposals.

## MVP (launched)

A Colombian SME connects WhatsApp, gets an out-of-box CRM for its vertical, an AI copilot that drafts replies (a human approves), full Ley 1581 compliance, and pays via PSE/Nequi.

**In MVP:**
- Stytch passwordless auth · org multi-tenancy
- WhatsApp ingress + inbox + CRM (contacts, companies, deals, pipelines, activities, tags)
- AI copilot: draft → human approve/reject → send (copilot mode only)
- Guardrails: PII masking, forbid/escalate rules, consent state machine
- Ley 1581: consent autoresponder, data export/forget
- 5 vertical playbooks + one-tap onboarding
- Tickets module (first sellable add-on)
- MercadoPago (PSE/Nequi) + Polar billing, provider routing
- AI usage metering + credit guard
- SSE streaming chat
- E2E suite + CI matrix

**Explicitly deferred from MVP:** autopilot autonomous sending, multi-model routing, prompt registry, semantic cache, evals, CRM business AI, copilot panel / command palette, OTel tracing, agent tool-use runtime.

---

## Phase Q2 — Trust & Depth

Goal: make the existing AI real and resilient; ship autonomous sending safely.

1. **Autopilot** — windowed autonomous WhatsApp replies gated by kill switch, consent, daily limits, and never/escalate guardrails; any block falls back to a copilot draft. (Deferred from the `add-agentic-whatsapp-assistant` change's tasks 7.x.)
2. **LLM provider router + fallback chain** — mirror the billing Polar→MercadoPago `ProviderRouter` pattern: route by task/budget, fail over primary→secondary→tertiary. Single-provider AI is the last un-hardened outbound seam in the codebase.
3. **AI usage/cost dashboards** — the `ai_usage` / `ai_usage_events` ledger and `GET /api/crm/usage/ai` already exist; surface credits used/remaining and cost per period in the UI.
4. **Semantic cache** — pgvector-based cache for similar queries (>0.95 cosine), targeting 30-50% cost reduction on RAG.
5. **RAGAS evals + regression suite** — automated retrieval/answer-quality scoring to catch prompt regressions before they reach tenants.

## Phase Q3 — Agent Becomes an Agent

Goal: AI that acts in the CRM, not just drafts replies.

1. **Tool registry + CRM tools** — extend the linear `flow_executor` with a tool loop (create deal, open ticket, query pipeline, update contact), with per-run iteration and cost limits.
2. **Scheduled / triggered workflows** — "when a lead is added, enrich and score it"; "when a deal is stuck, suggest next steps"; weekly pipeline reports.
3. **CRM business AI** — lead scoring with explanations, deal health prediction + next-best-action, conversation summaries.

## Phase Q4 — AI-Native UX & Scale Hygiene

Goal: AI becomes the primary interaction paradigm; engineering catches up.

1. **Copilot panel** — always-available context-aware assistant across CRM/inbox/settings.
2. **Inline AI + command palette** — AI field generation, anomaly highlights, natural-language → app action.
3. **OTel tracing + alerts** — LLM traces, latency/cost dashboards, SLO alerts.
4. **FE unit tests + full CI matrix** — frontend unit tests, E2E in CI, deploy pipeline.

## Beyond

- Self-improving AI: evals-driven prompt iteration, feedback loops, personalization, fine-tuning — only if data volume justifies it (likely premature).
- Vercel AI SDK (`@ai-sdk/react`) adoption for the streaming chat once network access is available (MVP shipped with a native fetch-stream consumer).

## Sequencing Notes

- Each phase above is a **capability change proposal**, created under `openspec/changes/` when work begins.
- Deferred tasks inside completed changes (e.g., autopilot 7.x in `add-agentic-whatsapp-assistant`) are the seeds of Q2 proposals; keep them referenced here rather than silently reopened.
