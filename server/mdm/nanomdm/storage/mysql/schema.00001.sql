ALTER TABLE devices ADD COLUMN bootstrap_token_b64 TEXT NULL;
ALTER TABLE devices ADD COLUMN bootstrap_token_at TIMESTAMP NULL;
ALTER TABLE devices ADD CONSTRAINT CHECK (bootstrap_token_b64 IS NULL OR bootstrap_token_b64 != '');
