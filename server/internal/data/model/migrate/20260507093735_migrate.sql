-- Modify "users" table
ALTER TABLE "users" ADD COLUMN "oauth_provider" character varying NULL, ADD COLUMN "oauth_subject" character varying NULL, ADD COLUMN "oauth_email" character varying NULL, ADD COLUMN "oauth_display_name" character varying NULL;
-- Create index "user_oauth_provider_oauth_subject" to table: "users"
CREATE UNIQUE INDEX "user_oauth_provider_oauth_subject" ON "users" ("oauth_provider", "oauth_subject");
