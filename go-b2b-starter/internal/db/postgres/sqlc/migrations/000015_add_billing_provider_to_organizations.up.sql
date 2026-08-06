-- Add billing_provider column to organizations for multi-provider routing
-- Supports: 'polar' (default), 'mercadopago'

ALTER TABLE organizations.organizations
ADD COLUMN billing_provider VARCHAR(50) DEFAULT NULL;

COMMENT ON COLUMN organizations.organizations.billing_provider IS 'Billing provider preference for this organization. NULL/polar = Polar.sh, mercadopago = MercadoPago';
