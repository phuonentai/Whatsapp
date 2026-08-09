// lib/api/api/dto/agent.dto.ts

export interface AgentSettingsDto {
  id?: number;
  organization_id: number;
  mode: string;
  tone: string;
  brand_voice?: string;
  autopilot_start?: string;
  autopilot_end?: string;
  timezone?: string;
  kill_switch: boolean;
  max_daily_messages: number;
  consent_required: boolean;
  consent_template?: string;
  guardrails: {
    never?: {
      max_discount_percent?: number;
      forbidden_terms?: string[];
    };
    escalate?: {
      terms?: string[];
    };
  };
  created_at?: string;
  updated_at?: string;
}

export interface AgentSuggestionDto {
  id: number;
  organization_id: number;
  conversation_id: number;
  contact_id: number;
  flow_id?: number;
  type: string;
  body: string;
  metadata?: Record<string, unknown>;
  status: string;
  source: string;
  approved_by_member_id?: string;
  whatsapp_message_id?: string;
  created_at: string;
  updated_at: string;
}

export interface ComplianceExportDto {
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
