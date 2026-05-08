-- Modify "gateway_api_keys" table
ALTER TABLE "gateway_api_keys" ADD COLUMN "plain_key" character varying NOT NULL DEFAULT '';
