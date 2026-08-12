-- 000034_create_conversation_contexts.up.sql
-- Per-conversation AI context cache (summary, intent, key facts).
-- One row per conversation; regenerated when new messages arrive.

CREATE TABLE agent.conversation_contexts (
    conversation_id INTEGER PRIMARY KEY REFERENCES crm.conversations(id) ON DELETE CASCADE,
    summary TEXT,
    key_facts JSONB NOT NULL DEFAULT '[]'::jsonb,
    detected_intent VARCHAR(255),
    source_cursor BIGINT NOT NULL DEFAULT 0,
    consent_gated BOOLEAN NOT NULL DEFAULT FALSE,
    generated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_conversation_contexts_generated
    ON agent.conversation_contexts(generated_at DESC);
