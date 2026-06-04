-- Modify "gateway_usage_logs" table
ALTER TABLE "gateway_usage_logs" ADD COLUMN "client_type" character varying NOT NULL DEFAULT 'other';
-- Create index "gatewayusagelog_client_type_created_at" to table: "gateway_usage_logs"
CREATE INDEX "gatewayusagelog_client_type_created_at" ON "gateway_usage_logs" ("client_type", "created_at");
