ALTER TABLE whatsapp.whatsapp_configs
  ADD COLUMN waba_id VARCHAR(100),
  ADD COLUMN access_token VARCHAR(500),
  ADD COLUMN api_version VARCHAR(20) NOT NULL DEFAULT 'v21.0',
  ADD COLUMN graph_api_url VARCHAR(255) NOT NULL DEFAULT 'https://graph.facebook.com';

COMMENT ON COLUMN whatsapp.whatsapp_configs.waba_id IS 'WhatsApp Business Account ID for outbound API calls';
COMMENT ON COLUMN whatsapp.whatsapp_configs.access_token IS 'Permanent access token for WhatsApp Cloud API authentication';
COMMENT ON COLUMN whatsapp.whatsapp_configs.api_version IS 'WhatsApp Cloud API version (default: v21.0)';
COMMENT ON COLUMN whatsapp.whatsapp_configs.graph_api_url IS 'Base URL for Graph API (default: https://graph.facebook.com)';
