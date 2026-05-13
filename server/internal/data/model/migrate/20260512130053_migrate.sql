-- Modify "gateway_usage_logs" table
ALTER TABLE "gateway_usage_logs" ADD COLUMN "diagnostic" jsonb NULL;
-- Create index "gatewayusagelog_upstream_error_type_created_at" to table: "gateway_usage_logs"
CREATE INDEX "gatewayusagelog_upstream_error_type_created_at" ON "gateway_usage_logs" ("upstream_error_type", "created_at");
