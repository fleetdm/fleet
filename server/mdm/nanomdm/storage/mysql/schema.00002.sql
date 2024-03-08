ALTER TABLE users ADD COLUMN user_authenticate TEXT NULL;
ALTER TABLE users ADD COLUMN user_authenticate_at TIMESTAMP NULL;
ALTER TABLE users ADD CONSTRAINT CHECK (user_authenticate IS NULL OR user_authenticate != '');
ALTER TABLE users ADD COLUMN user_authenticate_digest TEXT NULL;
ALTER TABLE users ADD COLUMN user_authenticate_digest_at TIMESTAMP NULL;
ALTER TABLE users ADD CONSTRAINT CHECK (user_authenticate_digest IS NULL OR user_authenticate_digest != '');
