CREATE TABLE nano_dep_names (
    name VARCHAR(255) NOT NULL,

    -- OAuth1 Tokens
    consumer_key        TEXT NULL,
	consumer_secret     TEXT NULL,
	access_token        TEXT NULL,
	access_secret       TEXT NULL,
	access_token_expiry TIMESTAMP NULL,

    -- Config
    config_base_url VARCHAR(255) NULL,

    -- Token PKI
    tokenpki_cert_pem TEXT NULL,
    tokenpki_key_pem  TEXT NULL,

    -- Syncer
    -- From Apple docs: "The string can be up to 1000 characters".
    syncer_cursor VARCHAR(1024) NULL,
    syncer_cursor_at TIMESTAMP NULL,

    -- Assigner
    assigner_profile_uuid    TEXT NULL,
    assigner_profile_uuid_at TIMESTAMP NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (name),

    CHECK (tokenpki_cert_pem IS NULL OR SUBSTRING(tokenpki_cert_pem FROM 1 FOR 27) = '-----BEGIN CERTIFICATE-----'),
    CHECK (tokenpki_key_pem IS NULL OR SUBSTRING(tokenpki_key_pem FROM 1 FOR  5) = '-----')
);
