-- Modify "gateway_usage_logs" table
ALTER TABLE "gateway_usage_logs" ADD COLUMN "client_ip" character varying NOT NULL DEFAULT '';
-- Create index "gatewayusagelog_client_ip_created_at" to table: "gateway_usage_logs"
CREATE INDEX "gatewayusagelog_client_ip_created_at" ON "gateway_usage_logs" ("client_ip", "created_at");
