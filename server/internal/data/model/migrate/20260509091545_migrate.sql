-- Modify "gateway_api_keys" table
ALTER TABLE "gateway_api_keys" ADD COLUMN "quota_daily_tokens" bigint NOT NULL DEFAULT 0, ADD COLUMN "quota_weekly_tokens" bigint NOT NULL DEFAULT 0;
