export interface InstagramConfigDto {
  id: number;
  organization_id: number;
  ig_user_id: string;
  ig_username?: string;
  fb_page_id?: string;
  access_token?: string;
  token_expires_at?: string;
  token_expiry_warning?: boolean;
  webhook_secret: string;
  verify_token: string;
  api_version?: string;
  graph_api_url?: string;
  is_active: boolean;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface InstagramWebhookHealthDto {
  last_24h: number;
  last_7d: number;
  total: number;
  by_status: Record<string, number>;
  last_error?: string;
  last_error_at?: string;
}
