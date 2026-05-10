-- Modify "gateway_usage_logs" table
ALTER TABLE "gateway_usage_logs" ADD COLUMN "session_id" character varying NOT NULL DEFAULT '';
-- Create index "gatewayusagelog_session_id_created_at" to table: "gateway_usage_logs"
CREATE INDEX "gatewayusagelog_session_id_created_at" ON "gateway_usage_logs" ("session_id", "created_at");
