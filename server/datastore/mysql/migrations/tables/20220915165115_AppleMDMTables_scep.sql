/* This schema for SCEP storage is a Fleet adaptation of the following schema:
 * https://github.com/jessepeterson/mysqlscepserver/blob/f1abaac10899fddbe80b6424470b418ce1a446c4/schema.sql
 */

/* Certificate serials must be generated before certificate issuance.
 * While it may seem somehwat wasteful to have a table just for this
 * purpose it offers the opportunity to LEFT JOIN against the
 * certificate table to find any serials that were generated but which
 * did not result in an accompanying certificate. I.e. some problem with
 * signing that certificate. The timestamp here could be used to look at
 * the logs for that case. */
CREATE TABLE IF NOT EXISTS scep_serials (
    serial BIGINT NOT NULL AUTO_INCREMENT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (serial)
);

CREATE TABLE IF NOT EXISTS scep_certificates (
    serial BIGINT NOT NULL,

    -- the name field should contain either the common name of the
    -- certificate or, if the CN is empty, the SHA-256 of the entire
    -- certificate DER bytes.
    name             VARCHAR(1024) NULL,
    not_valid_before DATETIME NOT NULL,
    not_valid_after  DATETIME NOT NULL,
    certificate_pem  TEXT NOT NULL,
    revoked          BOOLEAN NOT NULL DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (serial),

    FOREIGN KEY (serial)
        REFERENCES scep_serials (serial),

    CHECK (SUBSTRING(certificate_pem FROM 1 FOR 27) = '-----BEGIN CERTIFICATE-----'),
    CHECK (name IS NULL OR name != '')
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
