# AI-First SaaS Architecture Roadmap

## Current State Assessment

```
YOUR AI MATURITY:  ~20% of what an AI-first SaaS needs

    [■■■■□□□□□□□□□□□□□□□□]  20% — Basic RAG is table stakes now

Currently you have:
  ✅ Document ingestion + OCR
  ✅ Vector embeddings + semantic search
  ✅ Basic RAG chat (single-turn, no streaming)
  ✅ Single LLM provider (OpenAI)
  ✅ Solid Clean Architecture + DI foundation

What "AI-first" means in 2026 is fundamentally different from "has AI features."
An AI-first app treats AI as the primary interaction paradigm, not a feature bolted on.
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        KNOWLEDGE BASE (RAG)                         │
│                                                                     │
│  ┌──────────┐    ┌──────────┐    ┌───────────┐    ┌─────────────┐  │
│  │  Upload  │───▶│   OCR    │───▶│ Embedding │───▶│  pgvector   │  │
│  │   PDF    │    │ (Mistral)│    │ (OpenAI)  │    │ 1536-dim    │  │
│  └──────────┘    └──────────┘    └───────────┘    └──────┬──────┘  │
│                                                          │         │
│  ┌──────────┐    ┌───────────┐    ┌───────────┐          │         │
│  │   Chat   │───▶│  Vector   │◀───│  RAG       │◀─────────┘         │
│  │  Input   │    │  Search   │    │  Service   │                    │
│  └──────────┘    └───────────┘    └─────┬─────┘                    │
│                                         │                           │
│                    ┌───────────┐         │                           │
│                    │   OpenAI  │◀────────┘                           │
│                    │  GPT Chat │                                    │
│                    └───────────┘                                    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Gap Analysis

### Dimension 1: AI as Primary UX Paradigm

```
CURRENT:  "Knowledge Base" is a sidebar item → isolated page
AI-FIRST: AI is everywhere — copilot panels, inline suggestions, autonomous workflows

┌─────────────────────────────────────────────────────────┐
│                     AI-FIRST UX MAP                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│   │ Copilot │  │  Inline │  │  Auto-  │  │  Voice/ │  │
│   │  Panel  │  │   AI    │  │  mation │  │  Multi- │  │
│   │ (always │  │ (fields,│  │ (AI     │  │  modal  │  │
│   │  there) │  │  tables)│  │  agents)│  │  input  │  │
│   └─────────┘  └─────────┘  └─────────┘  └─────────┘  │
│                                                         │
│   CURRENT: Only the chat box in Knowledge Base          │
└─────────────────────────────────────────────────────────┘
```

### Dimension 2: Model Intelligence Stack

```
                    YOU ARE HERE
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ Layer 1: RAG Chat ────── ✅ DONE                             │
│   Single-turn, one model, no streaming                       │
├──────────────────────────────────────────────────────────────┤
│ Layer 2: Streaming + Multi-model ──── ❌ MISSING             │
│   Real-time token streaming, model selection/routing         │
├──────────────────────────────────────────────────────────────┤
│ Layer 3: Agentic ──── ❌ MISSING                             │
│   Tool use, multi-step reasoning, function calling            │
├──────────────────────────────────────────────────────────────┤
│ Layer 4: Autonomous Workflows ──── ❌ MISSING                │
│   Scheduled agents, triggered workflows, decision engines    │
├──────────────────────────────────────────────────────────────┤
│ Layer 5: Self-improving ──── ❌ MISSING                      │
│   Fine-tuning, RLHF, eval-driven iteration, model distillation│
└──────────────────────────────────────────────────────────────┘
```

### Dimension 3: AI Infrastructure

| Capability | Current State | Target State |
|------------|--------------|--------------|
| Model providers | OpenAI only | Multi-provider with intelligent routing |
| Prompt management | Hardcoded strings | Versioned, A/B tested, with template engine |
| Observability | Token count only | Full tracing, cost dashboards, eval scores |
| Streaming | Platform has it (unused) | SSE across all AI endpoints |
| Vector store | pgvector (good) | pgvector + semantic cache |
| Guardrails | None | Content filtering, PII detection, prompt injection defense |
| Evaluation | None | Automated RAG evals, response quality scoring |
| Cost control | None | Per-user quotas, budget alerts, model tiering |
| Caching | None | Semantic cache for similar queries, response caching |
| Queue/Async | Basic event bus | Reliable job queue for async AI tasks |

### Dimension 4: AI-Powered Business Modules

```
CURRENT: Only "Knowledge Base" has AI

CRM (in-progress change)
  ❌ Lead scoring/prioritization
  ❌ AI-generated email drafts
  ❌ Meeting summarization
  ❌ Deal health prediction
  ❌ Next-best-action recommendations
  ❌ Contact enrichment

Billing
  ❌ Usage prediction
  ❌ Churn risk scoring
  ❌ Smart plan recommendations

Auth/Users
  ❌ Anomaly detection
  ❌ Smart onboarding
```

---

## The Roadmap: 5 Phases to AI-First

```
┌────────────┐   ┌────────────┐   ┌────────────┐   ┌────────────┐   ┌────────────┐
│  Phase 1   │   │  Phase 2   │   │  Phase 3   │   │  Phase 4   │   │  Phase 5   │
│  AI Infra  │──▶│ Streaming │──▶│  Agentic   │──▶│  AI-Native │──▶│    Self-   │
│ Foundation │   │ + Multi-  │   │   + CRM    │   │    UX      │   │  Improving │
│            │   │  Model    │   │   AI       │   │            │   │            │
│  ~4 weeks  │   │  ~6 weeks │   │  ~8 weeks  │   │  ~8 weeks  │   │  ~6 weeks  │
└────────────┘   └────────────┘   └────────────┘   └────────────┘   └────────────┘
```

---

### Phase 1: AI Infrastructure Foundation (~4 weeks)

**Goal**: Build the platform layer that everything else depends on. This is the highest-leverage work.

```
   PROMPT        │  EVAL        │  OBSER-
   MANAGEMENT    │  FRAMEWORK   │  VABILITY

┌──────────────┐ ┌────────────┐ ┌──────────────────┐
│ Prompt       │ │ RAGAS       │ │ OpenTelemetry    │
│ Registry +   │ │ evaluation  │ │ traces for every │
│ Versioning   │ │ harness     │ │ LLM call         │
│              │ │             │ │                  │
│ Templates    │ │ Automated   │ │ Cost tracking    │
│ with vars    │ │ quality     │ │ per-org/user     │
│              │ │ scoring     │ │                  │
│ A/B testing  │ │ Regression  │ │ Latency + token  │
│ framework    │ │ test suite  │ │ dashboards       │
└──────────────┘ └────────────┘ └──────────────────┘

   AI GATEWAY    │  GUARDRAILS  │  SEMANTIC CACHE
┌──────────────┐ ┌────────────┐ ┌──────────────────┐
│ Multi-model  │ │ PII/anonym  │ │ Cache similar    │
│ provider     │ │ ization     │ │ queries (>0.95)  │
│ routing      │ │             │ │                  │
│              │ │ Content     │ │ Embedding-based  │
│ Anthropic,   │ │ safety      │ │ cache key lookup │
│ Gemini, etc  │ │             │ │                  │
│              │ │ Prompt      │ │ 30-50% cost      │
│ Cost-based   │ │ injection   │ │ reduction        │
│ routing      │ │ defense     │ │                  │
└──────────────┘ └────────────┘ └──────────────────┘
```

**New Database Tables:**

```sql
-- Prompt registry
CREATE TABLE ai.prompts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    version       INT NOT NULL DEFAULT 1,
    template      TEXT NOT NULL,
    model         TEXT NOT NULL,
    params        JSONB DEFAULT '{}',
    metadata      JSONB DEFAULT '{}',
    is_active     BOOLEAN DEFAULT true,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(name, version)
);

-- Evaluation runs
CREATE TABLE ai.eval_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id     UUID REFERENCES ai.prompts(id),
    dataset_name  TEXT,
    scores        JSONB NOT NULL,
    traces        JSONB,
    model         TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Cost tracking
CREATE TABLE ai.api_costs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL,
    model         TEXT NOT NULL,
    tokens_in     INT NOT NULL,
    tokens_out    INT NOT NULL,
    cost_cents    DECIMAL(10,4) NOT NULL,
    endpoint      TEXT,
    cached        BOOLEAN DEFAULT false,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Semantic cache
CREATE TABLE ai.semantic_cache (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_hash    TEXT NOT NULL UNIQUE,
    query_embedding vector(1536),
    cached_response TEXT NOT NULL,
    model         TEXT NOT NULL,
    hit_count     INT DEFAULT 1,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    expires_at    TIMESTAMPTZ
);

CREATE INDEX idx_semantic_cache_embedding
    ON ai.semantic_cache USING ivfflat (query_embedding vector_cosine_ops) WITH (lists = 100);
```

**New Backend Modules:**

```
internal/platform/
  ├── prompt/
  │   ├── domain/
  │   │   ├── entity.go          # PromptTemplate, PromptVersion
  │   │   ├── repository.go      # PromptRepository interface
  │   │   └── errors.go
  │   ├── app/services/
  │   │   └── prompt_service.go  # Render, version, A/B test
  │   └── infra/
  │       └── prompt_repository.go
  │
  ├── eval/
  │   ├── domain/
  │   │   ├── entity.go          # EvalRun, EvalScores, EvalDataset
  │   │   └── service.go         # EvalRunner interface
  │   ├── app/services/
  │   │   └── eval_service.go    # RAGAS scoring, regression tests
  │   └── infra/
  │       └── ragas_evaluator.go
  │
  ├── observability/
  │   ├── domain/
  │   │   ├── entity.go          # TraceSpan, CostRecord, LatencyMetric
  │   │   └── service.go         # ObservabilityService interface
  │   └── infra/
  │       ├── otel_tracer.go     # OpenTelemetry integration
  │       └── cost_tracker.go    # Per-org cost aggregation
  │
  ├── guardrails/
  │   ├── domain/
  │   │   ├── entity.go          # GuardrailResult, Violation
  │   │   └── service.go         # GuardrailService interface
  │   └── infra/
  │       ├── pii_detector.go    # PII detection + anonymization
  │       ├── content_safety.go  # Harmful content filtering
  │       └── prompt_injection.go # Injection attack defense
  │
  └── cache/
      ├── domain/
      │   ├── entity.go          # CacheEntry, CacheStats
      │   └── service.go         # CacheService interface
      └── infra/
          └── semantic_cache.go  # pgvector-based semantic cache
```

**Key Decisions:**

| Decision | Choice | Rationale |
|----------|--------|-----------|
| AI Gateway pattern | Wrapper around existing `LLMClient` | Extends current architecture, no rewrites |
| Prompt storage | Database (not files) | Enables A/B testing, non-dev iteration |
| Observability | OpenTelemetry | Industry standard, vendor-agnostic |
| Semantic cache threshold | Cosine similarity > 0.95 | Balances accuracy vs cost savings |
| PII detection | Regex + NER model | Regex for structured PII, NER for unstructured |
| Guardrails placement | Pre/post-call middleware | Independent of business logic |

**Risks**: None significant — this is additive infrastructure that doesn't change existing behavior.

---

### Phase 2: Streaming + Multi-Model (~6 weeks)

**Goal**: Make AI feel instantaneous and give users/model choice.

```
STREAMING ARCHITECTURE

  Client                    Server                    LLM Provider
    │                         │                          │
    │  POST /cognitive/chat   │                          │
    │  { message, useRag }    │                          │
    │────────────────────────▶│                          │
    │                         │  Chat Completions        │
    │                         │  (stream: true)          │
    │                         │─────────────────────────▶│
    │                         │                          │
    │  SSE: token: "The"      │◀─── chunk 1 ────────────│
    │◀────────────────────────│                          │
    │  SSE: token: "revenue"  │◀─── chunk 2 ────────────│
    │◀────────────────────────│                          │
    │  SSE: token: " is"      │◀─── chunk 3 ────────────│
    │◀────────────────────────│                          │
    │  ...                    │                          │
    │  SSE: done              │◀─── [DONE] ─────────────│
    │◀────────────────────────│                          │
    │                         │                          │
    │  Also saves full        │                          │
    │  response to DB         │                          │

MULTI-MODEL ROUTING

  ┌──────────────────────────────────────────────────────┐
  │                    AI GATEWAY                         │
  │                                                       │
  │  ┌─────────────┐    ┌─────────────────────────────┐  │
  │  │   Router    │───▶│  Provider Selection          │  │
  │  │             │    │                             │  │
  │  │  Inputs:    │    │  Rule-based routing:        │  │
  │  │  - task     │    │  - Complex reasoning → Anth  │  │
  │  │  - budget   │    │  - Fast/cheap → Groq        │  │
  │  │  - speed    │    │  - Multimodal → Gemini       │  │
  │  │  - quality  │    │  - Fine-tune → Together      │  │
  │  │             │    │  - Default → OpenAI           │  │
  │  └─────────────┘    └─────────────────────────────┘  │
  │                                                       │
  │  FALLBACK CHAIN:                                     │
  │  Primary → Secondary → Tertiary                      │
  │  (timeout-based failover + error-type routing)       │
  └──────────────────────────────────────────────────────┘
```

**Backend Changes:**

```
internal/platform/llm/infra/
  ├── openai_client.go        # Add streaming to Complete()
  ├── anthropic_client.go     # NEW: Anthropic Claude provider
  ├── gemini_client.go        # NEW: Google Gemini provider
  ├── groq_client.go          # NEW: Groq (fast inference)
  ├── together_client.go      # NEW: Together AI (fine-tunes)
  └── router.go               # NEW: Multi-model intelligent routing
```

**Frontend Changes:**

```
next_b2b_starter/
  ├── lib/hooks/
  │   └── mutations/
  │       └── use-chat.ts     # Rewrite: streaming with Vercel AI SDK
  ├── app/dashboard/knowledge/components/
  │   ├── chat-interface.tsx  # Update: streaming token rendering
  │   └── chat-message.tsx    # Update: incremental token display
  └── components/
      └── model-selector.tsx  # NEW: Model selection dropdown
```

**New Database Tables:**

```sql
-- Model registry
CREATE TABLE ai.models (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      TEXT NOT NULL,       -- 'openai', 'anthropic', 'gemini', etc.
    model_id      TEXT NOT NULL,       -- 'gpt-4o', 'claude-opus-4', etc.
    display_name  TEXT NOT NULL,
    capabilities  JSONB DEFAULT '[]',  -- ['chat', 'embedding', 'vision', 'tools']
    cost_per_1k_in  DECIMAL(10,6),
    cost_per_1k_out DECIMAL(10,6),
    max_tokens    INT,
    is_active     BOOLEAN DEFAULT true
);

-- Model routing rules
CREATE TABLE ai.routing_rules (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type     TEXT NOT NULL,       -- 'chat', 'summarize', 'classify'
    priority      INT NOT NULL,
    conditions    JSONB NOT NULL,      -- budget, speed, quality requirements
    model_id      UUID REFERENCES ai.models(id),
    fallback_model_id UUID REFERENCES ai.models(id),
    is_active     BOOLEAN DEFAULT true
);
```

**Risks**: Streaming requires both backend SSE and frontend streaming consumers. Multi-model increases testing matrix. Need to handle provider-specific quirks (different API formats, rate limits, token counting).

---

### Phase 3: Agentic AI + CRM AI (~8 weeks)

**Goal**: AI that takes action, not just answers questions. CRM becomes the first AI-powered business module.

```
AGENTIC FRAMEWORK ARCHITECTURE

  ┌─────────────────────────────────────────────────────────┐
  │                   AI AGENT RUNTIME                       │
  │                                                          │
  │  ┌────────────────┐     ┌──────────────────────────┐   │
  │  │ Tool Registry  │     │    Reasoning Loop         │   │
  │  │                │     │                           │   │
  │  │ ┌────────────┐ │     │  ┌──────┐  ┌──────┐     │   │
  │  │ │ search_db  │ │◀───▶│  │ Think│─▶│ Act  │     │   │
  │  │ │ query_docs │ │     │  └──┬───┘  └──┬───┘     │   │
  │  │ │ send_email │ │     │     │         │         │   │
  │  │ │ create CRM │ │     │     ▼         ▼         │   │
  │  │ │ api_call   │ │     │  ┌──────┐  ┌────────┐  │   │
  │  │ │ ...        │ │     │  │Review│◀─│Observe │  │   │
  │  │ └────────────┘ │     │  └──────┘  └────────┘  │   │
  │  └────────────────┘     │                           │   │
  │                         │  Max iterations: 10       │   │
  │  ┌────────────────────┐ │  Cost limit: $0.50/run   │   │
  │  │   Memory System   │ └──────────────────────────┘   │
  │  │                   │                                  │
  │  │ Short-term:       │     ┌─────────────────────┐    │
  │  │  Conversation     │     │   Tool Execution    │    │
  │  │  window (10 msgs) │     │   Sandbox           │    │
  │  │                   │     │                     │    │
  │  │ Long-term:        │     │  - Timeouts (30s)   │    │
  │  │  Vector memory    │     │  - Retry (3x)       │    │
  │  │  per user/org     │     │  - Error handling   │    │
  │  │                   │     │  - Rollback         │    │
  │  │ Working:          │     │  - Audit log        │    │
  │  │  Scratchpad       │     └─────────────────────┘    │
  │  └────────────────────┘                                  │
  └─────────────────────────────────────────────────────────┘
```

**CRM AI Features:**

```
┌─────────────────────────────────────────────────────────────┐
│                     CRM AI CAPABILITIES                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  LEAD SCORING                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Auto-score leads on engagement, company fit,        │   │
│  │ behavior signals. AI explanation for each score.    │   │
│  │                                                     │   │
│  │ Score: 87/100                                       │   │
│  │ Reasons: Engaged with 3 emails this week,           │   │
│  │          Company is in target segment,              │   │
│  │          Visited pricing page twice                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  EMAIL GENERATION                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Draft personalized emails from contact context,     │   │
│  │ deal stage, conversation history.                   │   │
│  │ User-editable before sending.                       │   │
│  │                                                     │   │
│  │ "Write a follow-up email to John at Acme about      │   │
│  │  the proposal we sent last week"                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  DEAL HEALTH                                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Predict win probability, identify at-risk deals,    │   │
│  │ suggest next-best-action.                           │   │
│  │                                                     │   │
│  │ Deal: Acme Enterprise - $50K                        │   │
│  │ Health: ⚠️ At Risk (win prob: 34%)                   │   │
│  │ Reason: No activity in 14 days, champion left       │   │
│  │ Suggest: Schedule call with new stakeholder         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  AI COPILOT (CRM sidebar)                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ "Summarize this company's last 90 days of activity" │   │
│  │ "What should I focus on this week?"                 │   │
│  │ "Compare these two deals"                           │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  WORKFLOW AUTOMATION                                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Trigger → AI Action → Result                        │   │
│  │ "When a new lead is added, enrich and score it"     │   │
│  │ "When a deal is stuck, suggest next steps"          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Backend Modules to Create:**

```
internal/modules/
  ├── agents/
  │   ├── domain/
  │   │   ├── entity.go          # Agent, Tool, ToolCall, AgentRun
  │   │   ├── tool_registry.go   # ToolRegistry interface
  │   │   ├── agent_runtime.go   # AgentRuntime interface
  │   │   └── errors.go
  │   ├── app/services/
  │   │   ├── agent_runtime.go   # ReAct loop implementation
  │   │   └── tool_executor.go   # Safe tool execution with sandboxing
  │   └── infra/
  │       ├── tool_builtins.go   # Built-in tools (search, query, etc.)
  │       └── tool_crm.go        # CRM-specific tools
  │
  └── crm-ai/
      ├── domain/
      │   ├── entity.go          # LeadScore, EmailDraft, DealHealth, CopilotContext
      │   └── service.go         # CRM AI service interfaces
      ├── app/services/
      │   ├── lead_scorer.go     # Score and explain lead quality
      │   ├── email_drafter.go   # Generate contextual email drafts
      │   ├── deal_analyzer.go   # Deal health prediction + next actions
      │   └── copilot.go         # Context-aware CRM copilot
      └── infra/
          └── crm_tools.go       # CRM agent tools (read contacts, update deals, etc.)
```

**New Database Tables:**

```sql
-- Agent definitions
CREATE TABLE ai.agents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    description   TEXT,
    system_prompt TEXT NOT NULL,
    tools         JSONB DEFAULT '[]',  -- ['search_contacts', 'send_email', ...]
    max_iterations INT DEFAULT 10,
    cost_limit_cents INT DEFAULT 50,   -- $0.50 per run
    is_active     BOOLEAN DEFAULT true,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Agent run history
CREATE TABLE ai.agent_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id      UUID REFERENCES ai.agents(id),
    org_id        UUID NOT NULL,
    user_id       UUID NOT NULL,
    input         TEXT NOT NULL,
    output        TEXT,
    tool_calls    JSONB DEFAULT '[]',
    total_tokens  INT DEFAULT 0,
    total_cost_cents DECIMAL(10,4) DEFAULT 0,
    status        TEXT DEFAULT 'running',  -- running, completed, failed, timeout
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

-- Lead scores
CREATE TABLE crm.lead_scores (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id    UUID NOT NULL,
    score         INT NOT NULL,        -- 0-100
    factors       JSONB NOT NULL,      -- scoring factors and weights
    explanation   TEXT,                -- AI-generated explanation
    model         TEXT NOT NULL,
    scored_at     TIMESTAMPTZ DEFAULT NOW()
);

-- AI-generated emails
CREATE TABLE crm.ai_emails (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id    UUID NOT NULL,
    deal_id       UUID,
    subject       TEXT NOT NULL,
    body          TEXT NOT NULL,
    tone          TEXT DEFAULT 'professional',
    context       JSONB,              -- what was used to generate
    user_edit     TEXT,               -- user's edited version
    status        TEXT DEFAULT 'draft', -- draft, sent, discarded
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Deal health predictions
CREATE TABLE crm.deal_health (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id       UUID NOT NULL,
    win_probability DECIMAL(5,2),     -- 0.00 - 100.00
    health_status TEXT,               -- healthy, at_risk, critical
    risk_factors  JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',
    model         TEXT NOT NULL,
    predicted_at  TIMESTAMPTZ DEFAULT NOW()
);
```

**Risks**: Agents are hard — hallucinated actions, infinite loops, cost explosions. Need strong guardrails, execution timeouts, and per-run cost limits. CRM AI is the highest-value business differentiator but also the riskiest to get wrong.

---

### Phase 4: AI-Native UX (~8 weeks)

**Goal**: AI becomes the primary interface paradigm, not just a feature.

```
AI-NATIVE UX ARCHITECTURE

  ┌──────────────────────────────────────────────────────────────┐
  │  ┌──────────┐                           ┌──────────────────┐ │
  │  │          │       MAIN CONTENT        │    COPILOT       │ │
  │  │  Sidebar │                           │    PANEL         │ │
  │  │          │  ┌─────────────────────┐  │                  │ │
  │  │  Nav     │  │                     │  │  Context-aware   │ │
  │  │          │  │   Tables / Forms    │  │  AI assistance   │ │
  │  │  AI      │  │   / Dashboards      │  │                  │ │
  │  │  Actions │  │                     │  │  "Summarize      │ │
  │  │  Panel   │  │   with inline AI    │  │   this page"     │ │
  │  │          │  │   suggestions       │  │                  │ │
  │  │          │  │                     │  │  "What are the   │ │
  │  │          │  └─────────────────────┘  │   key metrics?"  │ │
  │  │          │                           │                  │ │
  │  └──────────┘                           └──────────────────┘ │
  └──────────────────────────────────────────────────────────────┘
```

**AI-Native UI Components:**

```
INLINE AI:
  - Smart form fills ("Generate description from notes")
  - Table row summaries
  - Anomaly highlights in dashboards
  - "Explain this" hover on any metric
  - Auto-complete with AI suggestions

COMMAND PALETTE AI:
  Natural language → App action

  "Show me deals closing this month"     → Filters + navigates
  "Create a contact for John at Acme"    → Opens pre-filled form
  "What's our revenue this quarter?"     → Shows dashboard
  "Email all leads who haven't responded" → Batch action

NOTIFICATION AI:
  - Smart digests ("Your weekly CRM summary")
  - Anomaly alerts ("Revenue dropped 20% this week")
  - "A deal went cold" → AI-suggested re-engagement email

WORKFLOW AUTOMATION:
  Trigger → AI Action → Result

  "When a new lead is added, enrich and score it"
  "When a deal is stuck, suggest next steps"
  "Every Monday, generate a pipeline report"
```

**Frontend Pattern (Vercel AI SDK):**

```typescript
// Streaming chat with tool use
const { messages, input, handleInputChange, handleSubmit } = useChat({
  api: '/api/cognitive/chat',
  maxToolRoundtrips: 5,
});

// Single completion for inline AI
const { completion } = useCompletion({
  api: '/api/cognitive/complete',
});

// Structured JSON for tool calls
const { object } = useObject({
  api: '/api/cognitive/object',
  schema: leadScoreSchema,
});
```

**New Components:**

```
next_b2b_starter/
  ├── components/
  │   ├── ai/
  │   │   ├── copilot-panel.tsx         # Always-visible sidebar copilot
  │   │   ├── inline-ai.tsx             # Inline AI suggestions
  │   │   ├── command-palette-ai.tsx     # Natural language command palette
  │   │   ├── ai-notification.tsx       # Smart notification rendering
  │   │   └── ai-action-button.tsx      # One-click AI actions
  │   └── ...
  └── app/dashboard/
      ├── layout.tsx                    # Update: Add copilot panel
      └── components/
          └── command-palette.tsx       # Update: Add AI commands
```

**Risks**: UX complexity explodes. Need careful information architecture to avoid AI fatigue. Context window management becomes critical — what data does the copilot have access to? Need to balance AI helpfulness vs intrusiveness.

---

### Phase 5: Self-Improving AI (~6 weeks)

**Goal**: AI that gets better over time based on usage data.

```
SELF-IMPROVING AI LOOP

  ┌───────────────────────────────────────────────────────────────┐
  │                                                               │
  │  User Feedback         Automated Evals                       │
  │  ┌───────────┐         ┌──────────────────────┐              │
  │  │ 👍 / 👎  │         │ RAGAS scores          │              │
  │  │ Ratings  │         │ Retrieval quality     │              │
  │  │ Edits    │         │ Answer accuracy       │              │
  │  │ Reports  │         │ Hallucination         │              │
  │  └─────┬─────┘         │ detection             │              │
  │        │               └────────┬─────────────┘              │
  │        └───────┬────────────────┘                             │
  │                ▼                                              │
  │      ┌─────────────────┐                                     │
  │      │  Prompt + Model │                                     │
  │      │   Optimization  │                                     │
  │      │  ─────────────  │                                     │
  │      │  A/B test       │                                     │
  │      │  Auto-prompt    │                                     │
  │      │  improvement    │                                     │
  │      │  Model selection│                                     │
  │      └─────────────────┘                                     │
  │                                                               │
  └───────────────────────────────────────────────────────────────┘

FINE-TUNING PIPELINE

  Collect high-quality examples
  → Validate + deduplicate
  → Train/RLHF
  → Evaluate against baseline
  → Deploy if better
  → Monitor post-deployment

PERSONALIZATION

  Per-user model behavior adaptation
  Learning from corrections
  Organization-specific knowledge base
  Custom embedding fine-tuning
```

**New Modules:**

```
internal/platform/
  ├── personalization/
  │   ├── domain/
  │   │   ├── entity.go          # UserPreference, OrgProfile, BehaviorPattern
  │   │   └── service.go         # PersonalizationService interface
  │   └── infra/
  │       └── behavior_tracker.go
  │
  └── finetune/
      ├── domain/
      │   ├── entity.go          # TrainingExample, FineTuneJob, ModelVersion
      │   └── service.go         # FineTuneService interface
      └── infra/
          └── openai_finetune.go # OpenAI fine-tuning API
```

**Risks**: Requires significant data volume. Fine-tuning may not be worth it for most SaaS use cases — prompt engineering + good RAG is often sufficient. Personalization needs careful privacy consideration.

---

## Priority Matrix

```
High Impact ─┐
             │  Streaming        AI-Native UX
             │  Multi-model      Copilot
             │  AI Gateway
             │
             │  Guardrails       Agents
             │  Semantic Cache   CRM AI
             │  Observability
             │  Eval Framework
             │                                   Fine-tuning
Low Impact ──┤  Prompt Mgmt      Workflows       Personalization
             │
             └────────────────────────────────────────────────
                Low Effort                        High Effort

    LEGEND:
    Phase 1 items   Phase 2   Phase 3   Phase 4   Phase 5
```

---

## Key Architectural Decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | **AI Gateway vs direct calls** | Wrapper around existing `LLMClient` | Extends current architecture, no rewrites. Build abstraction now — every future phase depends on it. |
| 2 | **Frontend streaming** | Vercel AI SDK | Industry standard, `useChat` + streaming + tool use patterns built-in. |
| 3 | **Prompt management** | Database (not files) | Enables A/B testing, non-dev iteration, versioning. |
| 4 | **Vector store** | pgvector (keep) | Fine for <10M vectors. Only migrate if scale requires it. |
| 5 | **Agent framework** | Custom runtime | Given Clean Architecture, a custom runtime that respects existing patterns > fighting LangChain. |
| 6 | **Guardrails placement** | Pre/post-call middleware | Independent of business logic, composable. |
| 7 | **Observability** | OpenTelemetry | Industry standard, vendor-agnostic. |

---

## Implementation Timeline

```
MONTH 1 ─────────────────────────────────────────────────────
  Week 1-2:  AI Gateway + Semantic Cache
  Week 3-4:  Prompt Registry + Observability + Cost Tracking

MONTH 2 ─────────────────────────────────────────────────────
  Week 5-6:  Guardrails (PII, Content Safety, Injection Defense)
  Week 7-8:  Eval Framework (RAGAS) + Regression Test Suite

MONTH 3-4 ───────────────────────────────────────────────────
  Week 9-10:  Streaming Backend (SSE)
  Week 11-12: Streaming Frontend (Vercel AI SDK)
  Week 13-14: Multi-Model Router + Provider Integrations
  Week 15-16: Model Selection UI + Fallback Chains

MONTH 5-6 ───────────────────────────────────────────────────
  Week 17-18: Agent Runtime Framework
  Week 19-20: CRM AI - Lead Scoring + Deal Health
  Week 21-22: CRM AI - Email Generation + Copilot
  Week 23-24: Agent Tool Registry + CRM Tools

MONTH 7-8 ───────────────────────────────────────────────────
  Week 25-26: Copilot Panel Component
  Week 27-28: Inline AI Components
  Week 29-30: Command Palette AI
  Week 31-32: Notification AI + Workflow Automation

MONTH 9 ─────────────────────────────────────────────────────
  Week 33-34: Self-Improving AI + Feedback Loops
  Week 35-36: Personalization + Fine-tuning Pipeline
```

---

## Quick Reference: New Database Tables by Phase

| Phase | Tables |
|-------|--------|
| **Phase 1** | `ai.prompts`, `ai.eval_runs`, `ai.api_costs`, `ai.semantic_cache` |
| **Phase 2** | `ai.models`, `ai.routing_rules` |
| **Phase 3** | `ai.agents`, `ai.agent_runs`, `crm.lead_scores`, `crm.ai_emails`, `crm.deal_health` |
| **Phase 4** | (no new tables — frontend components) |
| **Phase 5** | `ai.user_preferences`, `ai.training_examples`, `ai.finetune_jobs`, `ai.model_versions` |

---

## What Exists Today vs Target

| Capability | Current | Target |
|------------|---------|--------|
| LLM Providers | 1 (OpenAI) | 5+ with intelligent routing |
| Streaming | Platform only | End-to-end SSE |
| Prompt Management | Hardcoded | Versioned + A/B tested |
| Observability | Token count | Full tracing + cost dashboards |
| Guardrails | None | PII + content safety + injection defense |
| Evaluation | None | Automated RAGAS + regression tests |
| Caching | None | Semantic cache (30-50% cost savings) |
| Agents | None | ReAct framework with tool use |
| AI Features | 1 page (Knowledge Base) | AI everywhere (copilot, inline, automated) |
| CRM AI | None | Lead scoring, email gen, deal health, copilot |
| Self-improving | None | Feedback loops + eval-driven iteration |
| Cost Control | None | Per-org quotas + budget alerts |
