-- Modify "gateway_api_keys" table
ALTER TABLE "gateway_api_keys" ADD COLUMN "default_reasoning_effort" character varying NOT NULL DEFAULT '';
