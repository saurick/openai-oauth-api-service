-- Modify "gateway_api_keys" table
ALTER TABLE "gateway_api_keys" ADD COLUMN "upstream_strategy" character varying NOT NULL DEFAULT '';
