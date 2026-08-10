// DTOs for the campaign audience layer (/crm/campanas).

export interface SegmentFilter {
  field: string;
  op: string;
  value: unknown;
}

export interface SegmentDto {
  id: number;
  organization_id: number;
  nombre: string;
  filter_spec: SegmentFilter[];
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export type CampaignStatus = "draft" | "ready";

export interface CampaignDto {
  id: number;
  organization_id: number;
  nombre: string;
  segment_id: number;
  status: CampaignStatus;
  recipient_count: number;
  launched_at?: string | null;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface EvalResultDto {
  total: number;
  excluded_by_gates: number;
}

export interface AudienceBuildResultDto {
  filter_spec: SegmentFilter[];
  preview: EvalResultDto;
}

export type RecipientStatus = "pending" | "sent" | "failed" | "skipped";

export interface CampaignRecipientDto {
  id: number;
  campaign_id: number;
  contact_id: number;
  status: RecipientStatus;
  whatsapp_message_id?: string;
  error?: string;
  phone_number: string;
  display_name?: string;
  created_at: string;
  updated_at: string;
}
