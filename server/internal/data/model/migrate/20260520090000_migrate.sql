-- Do not retain downstream API key plaintext after one-time creation display.
UPDATE "gateway_api_keys" SET "plain_key" = '' WHERE "plain_key" <> '';
