-- 000018_create_ai_usage_ledger.down.sql

ALTER TABLE subscription_billing.quota_tracking DROP COLUMN IF EXISTS ai_credits_max;
DROP TABLE IF EXISTS subscription_billing.ai_usage_events;
DROP TABLE IF EXISTS subscription_billing.ai_usage;
