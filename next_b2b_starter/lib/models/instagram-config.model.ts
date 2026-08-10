export interface InstagramConfig {
  id: number;
  organizationId: number;
  igUserId: string;
  igUsername?: string;
  fbPageId?: string;
  accessToken?: string;
  tokenExpiresAt?: string;
  tokenExpiryWarning?: boolean;
  webhookSecret: string;
  verifyToken: string;
  apiVersion?: string;
  graphApiUrl?: string;
  isActive: boolean;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface InstagramConfigInput {
  ig_user_id: string;
  ig_username?: string;
  fb_page_id?: string;
  access_token?: string;
  token_expires_at?: string;
  webhook_secret?: string;
  verify_token?: string;
  api_version?: string;
  graph_api_url?: string;
  metadata?: Record<string, unknown>;
}
