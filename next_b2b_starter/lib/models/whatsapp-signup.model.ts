export interface WhatsAppSignupMetaConfig {
  app_id: string;
  config_id: string;
  redirect_uri?: string;
}

export type WhatsAppSignupStatusValue =
  | "exchanging"
  | "registering"
  | "verifying"
  | "connected"
  | "failed";

export interface WhatsAppSignupStatus {
  id: number;
  organization_id: number;
  status: WhatsAppSignupStatusValue;
  step?: string;
  error_code?: string;
  retry_count: number;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface WhatsAppSignupResult {
  status: WhatsAppSignupStatusValue;
  error_code?: string;
  config?: {
    phone_number_id: string;
    business_phone: string;
    app_id?: string;
    waba_id?: string;
    is_active: boolean;
  };
}
