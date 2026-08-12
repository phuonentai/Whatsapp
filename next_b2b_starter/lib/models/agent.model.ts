// lib/models/agent.model.ts

export type AgentMode = "copilot" | "autopilot";
export type AgentTone = "formal" | "casual";
export type SuggestionStatus = "pending" | "approved" | "rejected" | "superseded";
export type SuggestionSource = "copilot" | "autopilot_fallback" | "escalation";
export type SuggestionType = "reply" | "escalation";

export interface GuardrailRuleInput {
  never?: {
    max_discount_percent?: number;
    forbidden_terms?: string[];
  };
  escalate?: {
    terms?: string[];
  };
}

export interface AgentSettings {
  id?: number;
  organization_id: number;
  mode: AgentMode;
  tone: AgentTone;
  brand_voice?: string;
  autopilot_start?: string;
  autopilot_end?: string;
  timezone?: string;
  kill_switch: boolean;
  max_daily_messages: number;
  consent_required: boolean;
  consent_template?: string;
  guardrails: GuardrailRuleInput;
}

export interface AgentSuggestion {
  id: number;
  organization_id: number;
  conversation_id: number;
  contact_id: number;
  flow_id?: number;
  type: SuggestionType;
  body: string;
  status: SuggestionStatus;
  source: SuggestionSource;
  approved_by_member_id?: string;
  whatsapp_message_id?: string;
  created_at: string;
  updated_at: string;
}

export interface ComplianceExport {
  contact: {
    phone_number: string;
    display_name?: string;
    email?: string;
    tipo_documento?: string;
    numero_documento?: string;
    consent_status: string;
    consented_at?: string;
  };
  conversations: Array<{
    id: number;
    status: string;
    messages: Array<{
      direction: string;
      message_type: string;
      content: string;
      status: string;
      created_at: string;
    }>;
  }>;
}

export type ConversationContextStatus = "available" | "unavailable" | "structural";

export interface ConversationContext {
  conversationId: number;
  summary?: string;
  detectedIntent?: string;
  keyFacts: string[];
  sourceCursor: number;
  generatedAt?: string;
  consentGated: boolean;
  status: ConversationContextStatus;
  channel?: string;
  messageCount?: number;
  firstMessageAt?: string;
  lastMessageAt?: string;
}

export type RephraseMode = "rephrase" | "formal" | "casual" | "summarize";

export interface RephraseResponse {
  text: string;
}
