-- Agent module queries (agentic WhatsApp assistant)

-- name: CreateConversationFlow :one
INSERT INTO agent.conversation_flows (organization_id, conversation_id, contact_id, status)
VALUES ($1, $2, $3, 'running')
RETURNING *;

-- name: GetConversationFlow :one
SELECT * FROM agent.conversation_flows
WHERE id = $1 AND organization_id = $2;

-- name: GetActiveFlowByConversation :one
SELECT * FROM agent.conversation_flows
WHERE organization_id = $1
  AND conversation_id = $2
  AND status IN ('running', 'awaiting_human')
ORDER BY id DESC
LIMIT 1;

-- name: UpdateFlowStatus :one
UPDATE agent.conversation_flows
SET status = $3, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: GetAgentSettings :one
SELECT * FROM agent.agent_settings
WHERE organization_id = $1;

-- name: UpsertAgentSettings :one
INSERT INTO agent.agent_settings (
    organization_id,
    mode,
    tone,
    brand_voice,
    autopilot_start,
    autopilot_end,
    timezone,
    kill_switch,
    max_daily_messages,
    consent_required,
    consent_template,
    guardrails
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (organization_id) DO UPDATE SET
    mode = EXCLUDED.mode,
    tone = EXCLUDED.tone,
    brand_voice = EXCLUDED.brand_voice,
    autopilot_start = EXCLUDED.autopilot_start,
    autopilot_end = EXCLUDED.autopilot_end,
    timezone = EXCLUDED.timezone,
    kill_switch = EXCLUDED.kill_switch,
    max_daily_messages = EXCLUDED.max_daily_messages,
    consent_required = EXCLUDED.consent_required,
    consent_template = EXCLUDED.consent_template,
    guardrails = EXCLUDED.guardrails,
    updated_at = NOW()
RETURNING *;

-- name: InsertSuggestion :one
INSERT INTO agent.agent_suggestions (
    organization_id,
    conversation_id,
    contact_id,
    flow_id,
    type,
    body,
    metadata,
    status,
    source,
    approved_by_member_id,
    whatsapp_message_id,
    request_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9, $10, $11
) RETURNING *;

-- name: ListSuggestionsByOrgStatus :many
SELECT * FROM agent.agent_suggestions
WHERE organization_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSuggestionByID :one
SELECT * FROM agent.agent_suggestions
WHERE id = $1 AND organization_id = $2;

-- name: ApproveSuggestion :one
UPDATE agent.agent_suggestions
SET status = 'approved', approved_by_member_id = $3, updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'pending'
RETURNING *;

-- name: RejectSuggestion :one
UPDATE agent.agent_suggestions
SET status = 'rejected', updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'pending'
RETURNING *;

-- name: GetPendingSuggestionByWhatsAppMessage :one
SELECT * FROM agent.agent_suggestions
WHERE organization_id = $1 AND whatsapp_message_id = $2 AND status = 'pending';

-- name: SupersedePendingSuggestionsForConversation :exec
UPDATE agent.agent_suggestions
SET status = 'superseded', updated_at = NOW()
WHERE organization_id = $1 AND conversation_id = $2 AND status = 'pending';

-- name: InsertAgentAction :one
INSERT INTO agent.agent_actions (
    organization_id,
    flow_id,
    action,
    decision,
    policy_input,
    reasons,
    approved_by_member_id,
    whatsapp_message_id,
    request_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: CountSentTodayByOrganization :one
SELECT COUNT(*) AS total
FROM crm.messages
WHERE organization_id = $1
  AND direction = 'outbound'
  AND created_at >= $2;

-- ============================================================
-- Compliance (Ley 1581): consent + export/forget on crm.contacts
-- ============================================================

-- name: UpdateContactConsent :one
UPDATE crm.contacts
SET consent_status = $3, consented_at = $4, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: AnonymizeContact :exec
UPDATE crm.contacts
SET display_name = '[ANONIMIZADO]',
    phone_number = '[ELIMINADO]',
    email = NULL,
    avatar_url = NULL,
    tipo_documento = NULL,
    numero_documento = NULL,
    consent_status = 'withdrawn',
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2;

-- name: ListConversationsByContact :many
SELECT * FROM crm.conversations
WHERE organization_id = $1 AND contact_id = $2
ORDER BY created_at DESC;
