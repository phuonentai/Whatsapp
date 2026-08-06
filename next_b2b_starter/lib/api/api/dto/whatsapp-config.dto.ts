export interface WhatsAppConfigDto {
  id: number;
  organization_id: number;
  phone_number_id: string;
  business_phone: string;
  webhook_secret: string;
  verify_token: string;
  app_id?: string;
  waba_id?: string;
  access_token?: string;
  api_version?: string;
  graph_api_url?: string;
  is_active: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}
