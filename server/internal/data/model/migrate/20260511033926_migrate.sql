-- Modify "gateway_usage_logs" table
ALTER TABLE "gateway_usage_logs" ADD COLUMN "reasoning_effort" character varying NOT NULL DEFAULT '';
-- Create index "gatewayusagelog_reasoning_effort_created_at" to table: "gateway_usage_logs"
CREATE INDEX "gatewayusagelog_reasoning_effort_created_at" ON "gateway_usage_logs" ("reasoning_effort", "created_at");
