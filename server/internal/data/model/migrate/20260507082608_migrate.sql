-- Modify "admin_users" table
ALTER TABLE "admin_users" ADD COLUMN "oauth_provider" character varying NULL, ADD COLUMN "oauth_subject" character varying NULL, ADD COLUMN "oauth_email" character varying NULL, ADD COLUMN "oauth_display_name" character varying NULL;
-- Create index "adminuser_oauth_provider_oauth_subject" to table: "admin_users"
CREATE UNIQUE INDEX "adminuser_oauth_provider_oauth_subject" ON "admin_users" ("oauth_provider", "oauth_subject");
