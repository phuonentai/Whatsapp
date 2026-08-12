// WhatsApp message template model (mirrors whatsapp.templates row).

export type TemplateStatus =
  | "draft"
  | "submitted"
  | "approved"
  | "rejected"
  | "paused";

export interface WhatsAppTemplate {
  id: number;
  organization_id: number;
  name: string;
  category: string;
  language: string;
  body: string;
  param_count: number;
  status: TemplateStatus;
  meta_template_id: string | null;
  rejection_reason: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface WhatsAppTemplateInput {
  name: string;
  category: string;
  language: string;
  body: string;
}

export interface TemplateSendInput {
  template_id: number;
  params: string[];
}
