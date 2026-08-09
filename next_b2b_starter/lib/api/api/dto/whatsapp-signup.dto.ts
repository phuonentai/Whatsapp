export interface WhatsAppSignupMetaConfigDto {
  app_id: string;
  config_id: string;
  redirect_uri?: string;
}

export interface WhatsAppSignupStatusDto {
  id: number;
  organization_id: number;
  status: string;
  step?: string;
  error_code?: string;
  retry_count: number;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface WhatsAppSignupResultDto {
  status: string;
  error_code?: string;
  config?: {
    phone_number_id: string;
    business_phone: string;
    app_id?: string;
    waba_id?: string;
    is_active: boolean;
  };
}

export interface ExchangeSignupRequestDto {
  code: string;
}
