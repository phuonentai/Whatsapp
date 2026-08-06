export interface WhatsAppConfig {
  id: number;
  organizationId: number;
  phoneNumberId: string;
  businessPhone: string;
  webhookSecret: string;
  verifyToken: string;
  appId?: string;
  wabaId?: string;
  accessToken?: string;
  apiVersion?: string;
  graphApiUrl?: string;
  isActive: boolean;
  metadata?: Record<string, unknown>;
  createdAt: Date;
  updatedAt: Date;
}

export interface WhatsAppConfigInput {
  phone_number_id: string;
  business_phone: string;
  webhook_secret?: string;
  verify_token?: string;
  app_id?: string;
  waba_id?: string;
  access_token?: string;
  api_version?: string;
  graph_api_url?: string;
  metadata?: Record<string, unknown>;
}
